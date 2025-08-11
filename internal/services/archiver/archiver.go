package archiver

import (
	"context"
	"errors"
	"sync/atomic"

	object_storage "github.com/fandasy/06.08.2025/internal/object-storage"
	fast_id "github.com/fandasy/06.08.2025/pkg/fast-id"

	"github.com/google/uuid"
)

var (
	ErrMaxTasksExceeded   = errors.New("max tasks exceeded")
	ErrTaskNotFound       = errors.New("task not found")
	ErrNoObjectsToArchive = errors.New("no objects to archive")
	ErrServiceStopped     = errors.New("archiver service stopped")
)

// NewTask return error:
//   - ErrServiceStopped
//   - ErrMaxTasksExceeded
func (a *archiver) NewTask() (string, error) {
	if a.isStopped() {
		return "", ErrServiceStopped
	}

	if !incrementWithMax(&a.active, a.cfg.MaxTasks) {
		return "", ErrMaxTasksExceeded
	}

	id := newID()
	t := newTask(id, a.cfg.MaxObjects)

	a.mu.Lock()
	a.tasks[id] = t
	a.mu.Unlock()

	return id, nil
}

// AddObjects return error:
//   - ErrServiceStopped
//   - ErrTaskNotFound
//   - ErrTaskInProgress
//   - ErrTaskCompleted
func (a *archiver) AddObjects(id string, urls []string) (int, error) {
	if a.isStopped() {
		return 0, ErrServiceStopped
	}

	a.mu.RLock()
	t, ok := a.tasks[id]
	a.mu.RUnlock()
	if !ok {
		return 0, ErrTaskNotFound
	}

	toAdd, ready, err := t.AddObjects(urls, a.cfg.MaxObjects)
	if err != nil {
		return 0, err
	}

	if ready {
		a.wg.Add(1)
		go a.processTask(t)
	}

	return toAdd, nil
}

// GetStatus return error:
//   - ErrServiceStopped
//   - ErrTaskNotFound
func (a *archiver) GetStatus(id string) (*TaskInfo, error) {
	if a.isStopped() {
		return nil, ErrServiceStopped
	}

	a.mu.RLock()
	t, ok := a.tasks[id]
	a.mu.RUnlock()
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t.Info(), nil
}

func (a *archiver) processTask(t *task) {
	defer a.active.Add(^uint32(0))
	defer a.wg.Done()

	var toSave []*object_storage.ArchiveObject

	for i, obj := range t.Objects() {
		archObj, err := a.getter.ToLink(obj.src)
		if err != nil {
			t.setObjectError(i, err)
			continue
		}
		toSave = append(toSave, archObj)
	}

	if len(toSave) == 0 {
		t.fail(ErrNoObjectsToArchive)
		return
	}

	link, err := a.saver.SaveArchive(t.id, toSave)
	if err != nil {
		t.fail(err)
		return
	}

	t.complete(link)
}

func (a *archiver) Stop(ctx context.Context) error {
	if a.isStopped() {
		return ErrServiceStopped
	}

	a.stopOnce.Do(func() { close(a.stopCh) })

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (a *archiver) isStopped() bool {
	select {
	case <-a.stopCh:
		return true
	default:
		return false
	}
}

func incrementWithMax(a *atomic.Uint32, Max uint32) bool {
	for {
		current := a.Load()
		if current == Max {
			return false
		}
		if a.CompareAndSwap(current, current+1) {
			break
		}
	}

	return true
}

func newID() string {
	var id string

	UUID, err := uuid.NewRandom()
	if err != nil {
		id = fast_id.New()
	} else {
		id = UUID.String()
	}

	return id
}
