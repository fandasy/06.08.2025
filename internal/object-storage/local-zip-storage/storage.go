package local_zip_storage

import (
	"archive/zip"
	object_storage "github.com/fandasy/06.08.2025/internal/object-storage"
	"github.com/fandasy/06.08.2025/pkg/e"
	"os"
	"path"
)

type Storage struct {
	addr string
	dir  string
}

func New(addr string, dir string) *Storage {
	return &Storage{dir: dir}
}

func (s *Storage) SaveArchive(name string, objects []*object_storage.ArchiveObject) (string, error) {
	localPath := path.Join(s.dir, name)

	zipFile, err := os.Create(localPath)
	if err != nil {
		return "", e.Wrap("local-zip-storage.os.Create", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, object := range objects {
		header := &zip.FileHeader{
			Name:     object.Name,
			Method:   zip.Deflate,
			Modified: object.Time,
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return "", e.Wrap("local-zip-storage.zip.CreateHeader", err)
		}

		if _, err := writer.Write(object.Content); err != nil {
			return "", e.Wrap("local-zip-storage.writer.Write", err)
		}
	}

	fullPath := path.Join(s.addr, localPath)

	return fullPath, nil
}
