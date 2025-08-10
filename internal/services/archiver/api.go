package archiver

import (
	"context"
	object_storage "github.com/fandasy/06.08.2025/internal/object-storage"
	"sync"
	"sync/atomic"
)

type Archiver interface {
	// NewTask return error:
	//  - ErrServiceStopped
	//  - ErrMaxTasksExceeded
	NewTask() (string, error)

	// AddObjects return error:
	//  - ErrServiceStopped
	//  - ErrTaskNotFound
	//  - ErrTaskInProgress
	//  - ErrTaskCompleted
	AddObjects(id string, urls []string) (int, error)

	// GetStatus return error:
	//  - ErrServiceStopped
	//  - ErrTaskNotFound
	GetStatus(id string) (*TaskInfo, error)

	Stop(ctx context.Context) error
}

type ArchiveObjectGetter interface {
	ToLink(link string, validContentTypes []string) (*object_storage.ArchiveObject, error)
}

type ArchiveSaver interface {
	SaveArchive(name string, objects []*object_storage.ArchiveObject) (string, error)
}

type archiver struct {
	cfg    Config
	getter ArchiveObjectGetter
	saver  ArchiveSaver

	mu    sync.RWMutex
	tasks map[string]*task

	active atomic.Uint32

	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

type Config struct {
	MaxTasks         uint32
	MaxObjects       int
	ValidContentType []string
}

func New(cfg Config, getter ArchiveObjectGetter, saver ArchiveSaver) Archiver {
	cfg.validate()

	return &archiver{
		cfg:    cfg,
		getter: getter,
		saver:  saver,
		tasks:  make(map[string]*task),
		stopCh: make(chan struct{}),
	}
}

const (
	defaultMaxTasks   = 3
	defaultMaxObjects = 3
)

var defaultValidContentType = []string{"text/plain"}

func (cfg *Config) validate() {
	if cfg.MaxTasks == 0 {
		cfg.MaxTasks = defaultMaxTasks
	}
	if cfg.MaxObjects <= 0 {
		cfg.MaxObjects = defaultMaxObjects
	}
	if cfg.ValidContentType == nil || len(cfg.ValidContentType) == 0 {
		cfg.ValidContentType = defaultValidContentType
	}
}
