package main

import (
	"bytes"
	"os"
	"path"
	"testing"
)

const (
	mockStoragePath = "./mocks/storage"
)

func TestNewStorage(t *testing.T) {
	cfg := StorageConfig{
		Path:    "./mocks/storage",
		MaxSize: 1000000,
		Limit:   10000000,
	}

	v := NewStorage(&cfg)
	if v == nil {
		t.Errorf("Res must not be nil")
	}
}

func TestStorageGetMaxSizeOfFile(t *testing.T) {
	cfg := StorageConfig{
		Path:    "./mocks/storage",
		MaxSize: 1000000,
		Limit:   10000000,
	}

	v := NewStorage(&cfg)
	size := v.GetMaxSizeOfFile()

	if size != cfg.MaxSize {
		t.Errorf("Size must be %d but got %d\n", cfg.MaxSize, size)
	}
}

func TestCreateFile(t *testing.T) {
	cases := []struct {
		cfg      StorageConfig
		hasError bool
		data     []byte
		hash     string
	}{
		{
			cfg: StorageConfig{
				Path:    mockStoragePath,
				MaxSize: 1000000,
				Limit:   10000000,
			},
			data:     []byte("example"),
			hasError: false,
			hash:     "example",
		},
		{
			cfg: StorageConfig{
				Path:    mockStoragePath,
				MaxSize: 1000000,
				Limit:   10000000,
			},
			hasError: true,
			hash:     "example",
		},
	}

	// remove example path before test
	defFile := path.Join(mockStoragePath, "ex", "example")
	if _, err := os.Stat(defFile); err == nil {
		os.Remove(defFile)
	}

	for _, tc := range cases {
		v := NewStorage(&tc.cfg)

		size, err := v.CreateFile(tc.hash, bytes.NewBuffer(tc.data))

		if tc.hasError {
			if err == nil {
				t.Error("Error must not be nil")
			}

			continue
		}

		if err != nil {
			t.Errorf("Error must be nil but got %v\n", err)
		}

		expectedSize := int64(len(tc.hash))
		if size != expectedSize {
			t.Errorf("Size must be %d but got %d\n", expectedSize, size)
		}
	}
}

func TestStorageGetFile(t *testing.T) {
	cases := []struct {
		name     string
		fileName string
		ok       bool
	}{
		{
			name:     "example",
			fileName: "",
			ok:       false,
		},
		{
			name:     "example",
			fileName: "mocks/storage/ex/example",
			ok:       true,
		},
	}

	cfg := StorageConfig{
		Path:    "./mocks/storage",
		MaxSize: 1000000,
		Limit:   10000000,
	}

	v := NewStorage(&cfg)

	for _, tc := range cases {
		fullName := path.Join(mockStoragePath, tc.name[:2], tc.name)

		if !tc.ok {
			if _, err := os.Stat(fullName); err == nil {
				os.Remove(fullName)
			}
		} else {
			if _, err := os.Stat(fullName); os.IsNotExist(err) {
				v.CreateFile(tc.name, bytes.NewBuffer([]byte(tc.name)))
			}
		}

		fileName, ok := v.GetFile(tc.name)

		if tc.ok != ok {
			t.Errorf("Ok must be %t but got %t\n", tc.ok, ok)
		}

		if tc.fileName != fileName {
			t.Errorf("Filename must be %s but got %s\n", tc.fileName, fileName)
		}
	}
}

func TestStorageGetFileSize(t *testing.T) {
	cases := []struct {
		name     string
		notFound bool
		isDir    bool
	}{
		{
			name:     "example",
			notFound: true,
		},
		{
			name:  "example",
			isDir: true,
		},
		{
			name: "example",
		},
	}

	cfg := StorageConfig{
		Path:    "./mocks/storage",
		MaxSize: 1000000,
		Limit:   10000000,
	}

	v := NewStorage(&cfg)

	for _, tc := range cases {
		fullName := path.Join(mockStoragePath, tc.name[:2], tc.name)

		if tc.notFound {
			if _, err := os.Stat(fullName); err == nil {
				os.Remove(fullName)
			}
		} else if tc.isDir {
			if _, err := os.Stat(fullName); err == nil {
				os.Remove(fullName)
			}

			os.MkdirAll(fullName, 0755)
		} else {
			if _, err := os.Stat(fullName); err == nil {
				os.Remove(fullName)
			}

			v.CreateFile(tc.name, bytes.NewBuffer([]byte(tc.name)))
		}

		size, err := v.GetFileSize(tc.name)

		if tc.notFound || tc.isDir {
			if err == nil {
				t.Error("Error must not be nil")
			}

			if size != 0 {
				t.Errorf("Size must be nil but got %d\n", size)
			}
			continue
		}

		if err != nil {
			t.Errorf("Error must not nil but got %v\n", err)
		}

		expectedSize := int64(len(tc.name))
		if size != expectedSize {
			t.Errorf("Size must be %d but got %d\n", expectedSize, size)
		}

	}
}

func TestStorageRemoveFile(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{
			name: "example",
			ok:   false,
		},
		{
			name: "example",
			ok:   true,
		},
	}

	cfg := StorageConfig{
		Path:    "./mocks/storage",
		MaxSize: 1000000,
		Limit:   10000000,
	}

	v := NewStorage(&cfg)

	for _, tc := range cases {
		fullName := path.Join(mockStoragePath, tc.name[:2], tc.name)

		if !tc.ok {
			if _, err := os.Stat(fullName); err == nil {
				os.Remove(fullName)
			}
		} else {
			if _, err := os.Stat(fullName); os.IsNotExist(err) {
				v.CreateFile(tc.name, bytes.NewBuffer([]byte(tc.name)))
			}
		}

		ok, err := v.RemoveFile(tc.name)
		if tc.ok != ok {
			t.Errorf("Ok must be %t but got %t\n", tc.ok, ok)
		}

		if !ok {
			// here we could emulate only "not exists" error
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
			}
		} else {
			if _, err := os.Stat(fullName); err == nil {
				t.Error("File has not been removed")
			}
		}
	}
}
