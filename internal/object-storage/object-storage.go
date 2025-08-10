package object_storage

import (
	"time"
)

type ArchiveObject struct {
	Name    string
	Time    time.Time
	Content []byte
}
