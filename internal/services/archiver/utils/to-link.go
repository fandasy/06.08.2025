package utils

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	object_storage "github.com/fandasy/06.08.2025/internal/object-storage"
)

var (
	ErrFileNotFound           = errors.New("file not found")
	ErrIncorrectFormat        = errors.New("incorrect format")
	ErrBadRequest             = errors.New("bad request")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrAccessDenied           = errors.New("access denied")
	ErrInternalSourceError    = errors.New("internal source error")
)

type ArchiveObjectGetter struct {
	client *http.Client
}

func (a *ArchiveObjectGetter) ToLink(link string, contentTypes []string) (*object_storage.ArchiveObject, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err) // TODO
	}

	req.Close = true

	resp, err := a.client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}

	if err != nil {
		return nil, fmt.Errorf("request failed: %w, code: %d", err, resp.StatusCode)
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrFileNotFound

	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		location := resp.Header.Get("Location")
		if location == "" {
			return nil, errors.New("redirect with no Location header")
		}
		return a.ToLink(location, contentTypes)

	case http.StatusBadRequest:
		return nil, ErrBadRequest

	case http.StatusUnauthorized:
		return nil, ErrAuthenticationRequired

	case http.StatusForbidden:
		return nil, ErrAccessDenied

	case http.StatusTooManyRequests:
		return nil, ErrBadRequest

	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return nil, ErrInternalSourceError
	}

	contentType := resp.Header.Get("Content-Type")
	if !validateContentType(contentType, contentTypes) {
		return nil, fmt.Errorf("%w: %s", ErrIncorrectFormat, contentType)
	}

	filename := path.Base(link)
	if filename == "." || filename == "/" {
		filename = "file_" + time.Now().Format("20060102150405")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	return &object_storage.ArchiveObject{
		Name:    filename,
		Time:    time.Now(),
		Content: content,
	}, nil
}

func validateContentType(contentType string, validContentTypes []string) bool {
	for _, validContentType := range validContentTypes {
		if contentType == validContentType {
			return true
		}
	}

	return false
}
