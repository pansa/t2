package main

import (
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
)

const (
	metaPrefix = "META:"
	scoreKey   = "DOWNLOAD_SCORES"
)

// RedisConfig struct
type RedisConfig struct {
	Host string
	Port int
}

// FileMeta struct
type FileMeta struct {
	Hash      string
	CreatedAt *time.Time
	DeletedAt *time.Time
	Size      int64
	Score     int
}

// Redis struct
type Redis struct {
	*redis.Pool
	Config *RedisConfig
}

// SaveFileMeta method
func (r *Redis) SaveFileMeta(file *FileMeta) error {
	conn := r.Get()
	defer conn.Close()

	conn.Send("HMSET", metaPrefix+file.Hash, "size", file.Size, "created_at", file.CreatedAt.Unix())
	conn.Send("ZADD", scoreKey, file.Score, file.Hash)

	_, err := conn.Do("")

	return err
}

// IncScore method
func (r *Redis) IncScore(hash string) (int, error) {
	conn := r.Get()
	defer conn.Close()

	return redis.Int(conn.Do("ZINCRBY", scoreKey, 1, hash))
}

// GetUnusedFiles method
func (r *Redis) GetUnusedFiles(limit int) ([]string, error) {
	conn := r.Get()
	defer conn.Close()

	if limit < 1 {
		limit = 0
	} else {
		limit--
	}

	replies, err := redis.Values(conn.Do("ZRANGE", scoreKey, 0, limit))
	if err != nil {
		return nil, err
	}

	res := []string{}
	for _, v := range replies {
		hash, err := redis.String(v, nil)
		if err != nil {
			return nil, err
		}

		res = append(res, hash)
	}

	return res, nil
}

// MarkFileAsDeleted method
func (r *Redis) MarkFileAsDeleted(hash string, t *time.Time) error {
	conn := r.Get()
	defer conn.Close()

	size, err := redis.Int(conn.Do("ZSCORE", scoreKey, hash))
	if err != nil {
		return err
	}

	conn.Send("HMSET", metaPrefix+hash, "deleted_at", t.Unix(), "score", size)
	conn.Send("ZREM", scoreKey, hash)

	_, err = conn.Do("")

	return err
}

// NewRedis func returns Redis pointer
func NewRedis(cfg *RedisConfig) *Redis {
	// for simplicity we use default timeouts for connect/read/write and concrete values for idle/max clients
	// @todo move to config

	// default port is 6379
	if cfg.Port == 0 {
		cfg.Port = 6379
	}

	pool := &redis.Pool{
		MaxActive:   10,
		MaxIdle:     5,
		IdleTimeout: 10 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", cfg.Host+":"+strconv.Itoa(cfg.Port))
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &Redis{
		pool,
		cfg,
	}
}
