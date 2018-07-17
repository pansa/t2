package main

import (
	"os"
	"path"
	"time"
)

// Application struct contains
// Config pointer
type Application struct {
	Config    *Config
	Storage   *Storage
	RateLimit *RateLimit
	Redis     *Redis

	cleanInProgress bool
}

// AutoClean method
func (app *Application) AutoClean() error {
	if app.cleanInProgress || app.Config.Storage.Limit == 0 {
		return nil
	}

	app.cleanInProgress = true
	defer app.markCleanAsStopped()

	return app.autoClean()
}

func (app *Application) autoClean() error {
	if _, err := os.Stat(app.Config.Storage.Path); os.IsNotExist(err) {
		return err
	}

	dirs, err := getDirectories(app.Config.Storage.Path)
	if err != nil {
		return err
	}

	var size int64

	for _, dir := range dirs {
		s, err := getDirectorySize(path.Join(app.Config.Storage.Path, dir))
		if err != nil {
			return err
		}

		size += s
	}

	if size <= app.Config.Storage.Limit {
		return nil
	}

do:
	for {
		hashes, err := app.Redis.GetUnusedFiles(20)

		if err != nil {
			return err
		} else if len(hashes) == 0 {
			break
		}

		for _, hash := range hashes {
			fileSize, err := app.Storage.GetFileSize(hash)
			if err != nil {
				return err
			}

			deleted, err := app.Storage.RemoveFile(hash)
			if err != nil {
				return err
			}

			t := time.Now()

			err = app.Redis.MarkFileAsDeleted(hash, &t)
			if err != nil {
				return err
			}

			if deleted {
				size -= fileSize
			}

			if size <= app.Config.Storage.Limit {
				break do
			}
		}

	}

	return nil
}

func (app *Application) markCleanAsStopped() {
	app.cleanInProgress = false
}

// NewApplication func returns Application pointer
func NewApplication(cfg *Config, s *Storage, r *RateLimit, redis *Redis) *Application {
	return &Application{
		Config:    cfg,
		Storage:   s,
		RateLimit: r,
		Redis:     redis,
	}
}

func getDirectories(dirPath string) ([]string, error) {
	res := []string{}

	d, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	items, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if item.IsDir() {
			res = append(res, item.Name())
		}
	}

	return res, err
}

func getDirectorySize(dirPath string) (int64, error) {
	var size int64

	d, err := os.Open(dirPath)
	if err != nil {
		return 0, err
	}
	defer d.Close()

	items, err := d.Readdir(-1)
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		if !item.IsDir() {
			size += item.Size()
		}
	}

	return size, err
}
