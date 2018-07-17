package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
)

func TestNewRedis(t *testing.T) {
	cfg := RedisConfig{}

	r := NewRedis(&cfg)

	conn, err := r.Dial()

	if err != nil {
		t.Errorf("Could not connect to redis: %v\n", err)
		return
	}

	if conn == nil {
		t.Error("Connection must not be nil")
		return
	}

	defer conn.Close()

	err = r.TestOnBorrow(conn, time.Now())
	if err != nil {
		t.Errorf("Could not ping redis: %v\n", err)
	}
}

func TestRedisSaveMetaData(t *testing.T) {
	r := NewRedis(&RedisConfig{})

	// flush db before test (we can do it on test environment)
	conn := r.Get()
	defer conn.Close()
	conn.Do("FLUSHDB")

	createdAt := time.Now().Add(-1 * time.Minute)
	deletedAt := time.Now()

	data := FileMeta{
		Hash:      "example",
		CreatedAt: &createdAt,
		DeletedAt: &deletedAt,
		Size:      1024,
		Score:     11,
	}

	err := r.SaveFileMeta(&data)

	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	// get values from redis

	score, err := redis.Int(conn.Do("ZSCORE", scoreKey, data.Hash))
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	if score != data.Score {
		t.Errorf("Score must be %d but got %d\n", data.Score, score)
	}

	values, err := redis.Values(conn.Do("HMGET", metaPrefix+data.Hash, "size", "created_at", "deleted_at"))
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	for index, item := range values {
		switch index {
		case 0:
			val, err := redis.Int64(item, nil)
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
				continue
			}

			if val != data.Size {
				t.Errorf("Size must be %d but got %d\n", data.Size, val)
			}

		case 1:
			val, err := redis.Int64(item, nil)
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
				continue
			}

			if val != data.CreatedAt.Unix() {
				t.Errorf("CreatedAt must be %d but got %d\n", data.CreatedAt.Unix(), val)
			}

		case 2:
			// we could set deleted_at only on removing file

			_, err := redis.Int64(item, nil)
			if err != redis.ErrNil {
				t.Errorf("Error must be %v but got %v\n", redis.ErrNil, err)
			}
		}
	}
}

func TestRedisIncScore(t *testing.T) {
	r := NewRedis(&RedisConfig{})

	// flush db before test (we can do it on test environment)
	conn := r.Get()
	defer conn.Close()
	conn.Do("FLUSHDB")

	cases := []struct {
		hash  string
		score int
	}{
		{
			hash:  "example",
			score: 1,
		},
		{
			hash:  "example",
			score: 2,
		},
		{
			hash:  "example2",
			score: 1,
		},
		{
			hash:  "example",
			score: 3,
		},
		{
			hash:  "example2",
			score: 2,
		},
	}

	for _, tc := range cases {
		score, err := r.IncScore(tc.hash)

		if err != nil {
			t.Errorf("Error must be nil but got %v\n", err)
			continue
		}

		if score != tc.score {
			t.Errorf("Score must be %d but got %d\n", tc.score, score)
		}

		realScore, err := redis.Int(conn.Do("ZSCORE", scoreKey, tc.hash))
		if err != nil {
			t.Errorf("Error must be nil but got %v\n", err)
			continue
		}

		if realScore != tc.score {
			t.Errorf("Score must be %d but got %d\n", tc.score, realScore)
		}
	}
}

func TestRedisMarkFileAsDeleted(t *testing.T) {
	r := NewRedis(&RedisConfig{})

	// flush db before test (we can do it on test environment)
	conn := r.Get()
	defer conn.Close()
	conn.Do("FLUSHDB")

	createdAt := time.Now().Add(-1 * time.Minute)
	deletedAt := time.Now()

	data := FileMeta{
		Hash:      "example",
		CreatedAt: &createdAt,
		Size:      1024,
		Score:     11,
	}

	err := r.SaveFileMeta(&data)
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	err = r.MarkFileAsDeleted(data.Hash, &deletedAt)
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	val, err := redis.Int64(conn.Do("HGET", metaPrefix+data.Hash, "deleted_at"))
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	if val != deletedAt.Unix() {
		t.Errorf("DeletedAt must be %d but got %d\n", data.DeletedAt.Unix(), val)
	}

	score, err := redis.Int(conn.Do("HGET", metaPrefix+data.Hash, "score"))
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	if score != data.Score {
		t.Errorf("Score must be %d but got %d\n", data.Score, score)
	}

	_, err = redis.Int(conn.Do("ZSCORE", scoreKey, data.Hash))
	if err != redis.ErrNil {
		t.Errorf("Error must be %v but got %v\n", redis.ErrNil, err)
		return
	}
}

func TestRedisGetUnusedFiles(t *testing.T) {
	r := NewRedis(&RedisConfig{})

	// flush db before test (we can do it on test environment)
	conn := r.Get()
	defer conn.Close()
	conn.Do("FLUSHDB")

	createdAt := time.Now().Add(-1 * time.Minute)

	files := []FileMeta{
		FileMeta{
			Hash:      "example",
			CreatedAt: &createdAt,
			Size:      1024,
			Score:     100,
		},
		FileMeta{
			Hash:      "example2",
			CreatedAt: &createdAt,
			Size:      1024,
			Score:     101,
		},
		FileMeta{
			Hash:      "example3",
			CreatedAt: &createdAt,
			Size:      1024,
			Score:     200,
		},
		FileMeta{
			Hash:      "example4",
			CreatedAt: &createdAt,
			Size:      1024,
			Score:     50,
		},
		FileMeta{
			Hash:      "example5",
			CreatedAt: &createdAt,
			Size:      1024,
			Score:     30,
		},
		FileMeta{
			Hash:      "example6",
			CreatedAt: &createdAt,
			Size:      1024,
			Score:     0,
		},
	}

	for _, meta := range files {
		r.SaveFileMeta(&meta)
	}

	cases := []struct {
		limit    int
		expected []string
	}{
		{
			limit:    0,
			expected: []string{"example6"},
		},
		{
			limit:    1,
			expected: []string{"example6"},
		},
		{
			limit:    2,
			expected: []string{"example6", "example5"},
		},
		{
			limit:    3,
			expected: []string{"example6", "example5", "example4"},
		},
		{
			limit:    10,
			expected: []string{"example6", "example5", "example4", "example", "example2", "example3"},
		},
	}

	for _, tc := range cases {
		keys, err := r.GetUnusedFiles(tc.limit)
		if err != nil {
			t.Errorf("Error must be nil but got %v\n", err)
			continue
		}

		if !reflect.DeepEqual(keys, tc.expected) {
			t.Errorf("Keys must be %v but got %v\n", tc.expected, keys)
			continue
		}
	}

}
