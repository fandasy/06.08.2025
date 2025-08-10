package local_zip_storage

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	object_storage "github.com/fandasy/06.08.2025/internal/object-storage"
)

func TestSaveArchive_CreatesZipWithFiles(t *testing.T) {
	const storageDirName = "TEST"

	// tmpDir := t.TempDir()

	st, err := New("http://localhost/files", storageDirName)
	require.NoError(t, err)

	pdfPath := "./example-files/file.pdf"
	jpgPath := "./example-files/file.jpg"

	pdfData, err := os.ReadFile(pdfPath)
	require.NoError(t, err)

	jpgData, err := os.ReadFile(jpgPath)
	require.NoError(t, err)

	objects := []*object_storage.ArchiveObject{
		{
			Name:    path.Base(pdfPath),
			Time:    time.Now(),
			Content: pdfData,
		},
		{
			Name:    path.Base(jpgPath),
			Time:    time.Now(),
			Content: jpgData,
		},
	}

	zipName := "test.zip"
	fullPath, err := st.SaveArchive(zipName, objects)
	require.NoError(t, err)

	t.Log(fullPath)

	_, err = os.Stat(path.Join(storageDirName, zipName))
	require.NoError(t, err, "zip file not created")

	r, err := zip.OpenReader(path.Join(storageDirName, zipName))
	require.NoError(t, err)
	defer r.Close()

	require.Len(t, r.File, 2, "zip should contain 2 files")

	expected := map[string][]byte{
		"file.pdf": pdfData,
		"file.jpg": jpgData,
	}

	for _, f := range r.File {
		rc, err := f.Open()
		require.NoError(t, err)

		content, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)

		exp, ok := expected[f.Name]
		require.True(t, ok, "unexpected file in zip: %s", f.Name)
		require.True(t, bytes.Equal(exp, content), "file content mismatch for %s", f.Name)
	}
}
