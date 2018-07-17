package main

import (
	"bytes"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewApplication(t *testing.T) {
	app := NewApplication(&Config{}, &Storage{}, &RateLimit{}, &Redis{})

	if app.cleanInProgress {
		t.Errorf("cleanInProgress must be %t but got %t\n", false, app.cleanInProgress)
	}
}

func TestApplicationMarkCleanAsStopped(t *testing.T) {
	app := NewApplication(&Config{}, &Storage{}, &RateLimit{}, &Redis{})

	app.cleanInProgress = true

	app.markCleanAsStopped()

	if app.cleanInProgress {
		t.Errorf("cleanInProgress must be %t but got %t\n", false, app.cleanInProgress)
	}
}

func TestApplicationAutoClean(t *testing.T) {
	cases := []struct {
		limit  int64
		create bool
		redis  bool
	}{
		{
			limit: 0,
		},
		{
			limit: 1024,
		},
		{
			limit:  1024,
			create: true,
		},
		{
			limit:  1024,
			create: true,
			redis:  true,
		},
	}

	for _, tc := range cases {
		cfg := &StorageConfig{
			Path:  "mocks/storage/",
			Limit: tc.limit,
		}
		app := NewApplication(&Config{Storage: cfg}, NewStorage(cfg), NewRateLimit(&RateLimitConfig{}), NewRedis(&RedisConfig{}))

		// clear directory
		d, err := os.Open(cfg.Path + "ex")
		if err != nil {
			t.Errorf("Err must be nil but got %v\n", err)
			continue
		}

		items, err := d.Readdir(-1)
		d.Close()

		for _, item := range items {
			os.RemoveAll(path.Join(cfg.Path, "ex", item.Name()))
		}

		// flush redis database
		conn := app.Redis.Get()
		conn.Do("FLUSHDB")

		// now we're ready to test

		if tc.create {
			bytesCount := int(tc.limit / 4)
			total := rand.Intn(50) + 10
			for i := 0; i < total; i++ {
				hash := "example" + strconv.Itoa(i+1)
				buf := bytes.NewBuffer([]byte(strings.Repeat("a", bytesCount)))
				app.Storage.CreateFile(hash, buf)

				if tc.redis {
					t := time.Now()

					app.Redis.SaveFileMeta(&FileMeta{
						Hash:      hash,
						Size:      int64(bytesCount),
						CreatedAt: &t,
						Score:     rand.Intn(100),
					})
				}
			}
		}

		err = app.AutoClean()
		if err != nil {
			t.Errorf("Err must be nil but got %v\n", err)
		}

		// we could check size only if we stored meta data to redis
		if tc.limit > 0 && tc.redis {
			dirs, err := getDirectories(cfg.Path)
			if err != nil {
				t.Errorf("Err must be nil but got %v\n", err)
			} else {
				var size int64

				for _, dir := range dirs {
					s, err := getDirectorySize(path.Join(cfg.Path, dir))
					if err != nil {
						t.Errorf("Err must be nil but got %v\n", err)
					} else {
						size += s
					}
				}

				if size > cfg.Limit {
					t.Errorf("Size (%d) must be smaller than %d\n", size, cfg.Limit)
				}
			}
		}

		conn.Close()
	}
}

func TestGetDirectories(t *testing.T) {
	dirs, err := getDirectories("mocks")
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	for _, dir := range dirs {
		v, err := os.Stat(path.Join("mocks", dir))

		if err != nil {
			t.Errorf("Error must be nil but got %v\n", err)
			continue
		}

		if !v.IsDir() {
			t.Errorf("getDirectories must return only directories but got %#v\n", v)
		}
	}
}

func TestGetDirectorySize(t *testing.T) {
	size, err := getDirectorySize("mocks/config")
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	if size == 0 {
		t.Error("Size must be greater than 0")
	}
}
