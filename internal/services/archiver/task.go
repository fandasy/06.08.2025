package archiver

import (
	"errors"
	"sync"
)

type TaskStatus int8

const (
	_ TaskStatus = iota
	StatusWaitingForObjects
	StatusArchiving
	StatusDone
	StatusError
)

var (
	ErrTaskInProgress = errors.New("task already in progress")
	ErrTaskCompleted  = errors.New("task already completed")
)

type task struct {
	id string

	mu      sync.RWMutex
	status  TaskStatus
	objects []object

	zip string
	err error
}

type object struct {
	src string
	err error
}

func newTask(id string, maxObjects int) *task {
	return &task{
		id:      id,
		status:  StatusWaitingForObjects,
		objects: make([]object, 0, maxObjects),
	}
}

func (t *task) AddObjects(urls []string, maxObjects int) (int, bool, error) {
	if t.status == StatusWaitingForObjects {
		t.mu.Lock()
		defer t.mu.Unlock()

		free := maxObjects - len(t.objects)
		var toAdd int
		if len(t.objects) >= free {
			toAdd = free
		} else {
			toAdd = len(urls)
		}

		for i := 0; i < toAdd; i++ {
			t.objects = append(t.objects, object{src: urls[i]})
		}

		var ready bool

		if len(t.objects) == maxObjects {
			t.status = StatusArchiving
			ready = true
		}

		return toAdd, ready, nil

	} else {
		t.mu.RLock()
		defer t.mu.RUnlock()

		switch t.status {
		case StatusArchiving:
			return 0, false, ErrTaskInProgress

		case StatusDone, StatusError:
			return 0, false, ErrTaskCompleted

		default:
			return 0, false, nil
		}
	}
}

func (t *task) Objects() []object {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]object, len(t.objects))
	copy(out, t.objects)
	return out
}

func (t *task) setObjectError(objIndex int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.objects[objIndex].err = err
}

func (t *task) fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.err = err
	t.status = StatusError
}

func (t *task) complete(zip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.zip = zip
	t.status = StatusDone
}

type TaskInfo struct {
	Status  TaskStatus
	Objects []ObjectInfo
	Zip     string
	Err     error
}

type ObjectInfo struct {
	Src string
	Err error
}

func (t *task) Info() *TaskInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	objs := make([]ObjectInfo, 0, len(t.objects))
	for _, o := range t.objects {
		objs = append(objs, ObjectInfo{
			Src: o.src,
			Err: o.err,
		})
	}

	return &TaskInfo{
		Status:  t.status,
		Objects: objs,
		Zip:     t.zip,
		Err:     t.err,
	}
}

func (s TaskStatus) String() string {
	switch s {
	case StatusWaitingForObjects:
		return "Waiting for objects"
	case StatusArchiving:
		return "Archiving"
	case StatusDone:
		return "Done"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}
