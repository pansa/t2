package main

import (
	"errors"
	"io"
	"os"
	"path"
)

// StorageConfig struct contains info about
// - path which used for file saving
// - max size of file which can be uploaded on server
// - limit
type StorageConfig struct {
	Path    string `json:"path"`
	MaxSize int64  `json:"max_size"`
	Limit   int64  `json:"limit"`
}

// Storage struct
type Storage struct {
	Config *StorageConfig
}

// GetMaxSizeOfFile method return max file size in bytes
func (s *Storage) GetMaxSizeOfFile() int64 {
	return s.Config.MaxSize
}

// CreateFile method creates new file
func (s *Storage) CreateFile(hash string, b io.Reader) (int64, error) {
	folder := path.Join(s.Config.Path, hash[:2])

	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			return 0, err
		}
	}

	fileName := path.Join(folder, hash)

	if _, err := os.Stat(fileName); err == nil {
		return 0, errors.New("File already exists")
	}

	file, err := os.Create(fileName)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	bytesCount, err := io.Copy(file, b)

	if err != nil {
		return 0, err
	}

	err = file.Sync()
	if err != nil {
		return 0, err
	}

	return bytesCount, nil
}

// GetFile method returns content of file by hash
func (s *Storage) GetFile(hash string) (string, bool) {
	fileName := path.Join(s.Config.Path, hash[:2], hash)

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return "", false
	}

	return fileName, true
}

// GetFileSize method returns size of the file by hash
func (s *Storage) GetFileSize(hash string) (int64, error) {
	fileName := path.Join(s.Config.Path, hash[:2], hash)

	v, err := os.Stat(fileName)

	if err != nil {
		return 0, err
	}

	if v.IsDir() {
		return 0, errors.New("Not a file")
	}

	return v.Size(), nil
}

// RemoveFile method removes file from storage
func (s *Storage) RemoveFile(hash string) (bool, error) {
	fileName := path.Join(s.Config.Path, hash[:2], hash)

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return false, nil
	}

	err := os.Remove(fileName)
	if err != nil {
		return false, err
	}

	return true, nil
}

// NewStorage func returns Storage pointer
func NewStorage(cfg *StorageConfig) *Storage {
	return &Storage{
		Config: cfg,
	}
}
