package test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	object_storage "github.com/fandasy/06.08.2025/internal/object-storage"
	"github.com/fandasy/06.08.2025/internal/services/archiver"
)

type mockGetter struct {
	mu sync.Mutex
}

var ErrMockGetter = errors.New("mock getter error")

func (m *mockGetter) ToLink(link string) (*object_storage.ArchiveObject, error) {
	if link == "fail" {
		return nil, ErrMockGetter
	}
	return &object_storage.ArchiveObject{
		Name:    link,
		Time:    time.Now(),
		Content: []byte("data"),
	}, nil
}

type mockSaver struct {
	saved map[string][]*object_storage.ArchiveObject
	mu    sync.Mutex
}

func (m *mockSaver) SaveArchive(name string, objects []*object_storage.ArchiveObject) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saved == nil {
		m.saved = make(map[string][]*object_storage.ArchiveObject)
	}

	// Work
	time.Sleep(500 * time.Millisecond)

	m.saved[name] = objects
	return "http://test/" + name + ".zip", nil
}

func newTestArchiver(maxTasks uint32, maxObjects int) archiver.Archiver {
	cfg := archiver.Config{
		MaxTasks:   maxTasks,
		MaxObjects: maxObjects,
	}
	return archiver.New(cfg, &mockGetter{}, &mockSaver{})
}

func TestNewTaskAndGetStatus(t *testing.T) {
	a := newTestArchiver(3, 3)

	id, err := a.NewTask()
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	info, err := a.GetStatus(id)
	require.NoError(t, err)
	assert.Equal(t, archiver.StatusWaitingForObjects, info.Status)
	assert.Empty(t, info.Zip)
}

func TestAddObjectsTriggersArchive(t *testing.T) {
	a := newTestArchiver(3, 3)

	id, _ := a.NewTask()
	_, err := a.AddObjects(id, []string{"file1", "file2"})
	require.NoError(t, err)

	info, _ := a.GetStatus(id)
	assert.Equal(t, 2, len(info.Objects))
	assert.Equal(t, archiver.StatusWaitingForObjects, info.Status)

	// Trigger to work
	_, err = a.AddObjects(id, []string{"file3"})
	require.NoError(t, err)

	// Waiting for work to be completed
	time.Sleep(1 * time.Second)

	info, _ = a.GetStatus(id)
	assert.Equal(t, archiver.StatusDone, info.Status)
	assert.Contains(t, info.Zip, ".zip")
}

func TestMaxTasksExceeded(t *testing.T) {
	a := newTestArchiver(1, 3) // max 1 task

	id1, _ := a.NewTask()
	_, _ = a.AddObjects(id1, []string{"a", "b", "c"})

	// Expecting error: ErrMaxTasksExceeded
	_, err := a.NewTask()
	assert.ErrorIs(t, err, archiver.ErrMaxTasksExceeded)
}

func TestInvalidTaskOperations(t *testing.T) {
	a := newTestArchiver(3, 3)

	// Bad id
	_, err := a.AddObjects("bad-id", []string{"x"})
	assert.ErrorIs(t, err, archiver.ErrTaskNotFound)

	_, err = a.GetStatus("bad-id")
	assert.ErrorIs(t, err, archiver.ErrTaskNotFound)
}

func TestFailingGetter(t *testing.T) {
	getter := &mockGetter{}
	saver := &mockSaver{}
	cfg := archiver.Config{
		MaxTasks:   3,
		MaxObjects: 3,
	}
	a := archiver.New(cfg, getter, saver)

	id, _ := a.NewTask()
	_, _ = a.AddObjects(id, []string{"ok", "fail", "ok"})

	// Waiting for work to be completed
	time.Sleep(1 * time.Second)

	info, _ := a.GetStatus(id)
	assert.Equal(t, archiver.StatusDone, info.Status)

	// {"ok", "fail", "ok"}
	var failCount int
	for _, obj := range info.Objects {
		if obj.Err != nil {
			failCount++
		}
	}
	assert.Equal(t, 1, failCount)
}

func TestStopPreventsNewTasks(t *testing.T) {
	a := newTestArchiver(3, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := a.Stop(ctx)
	require.NoError(t, err)

	_, err = a.NewTask()
	assert.ErrorIs(t, err, archiver.ErrServiceStopped)
}
