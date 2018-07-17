package main

import (
	"sync"
	"time"
)

// RPSConfig config
type RPSConfig struct {
	Download int `json:"download"`
	Upload   int `json:"upload"`
	Remove   int `json:"remove"`
}

// BandwidthConfig config
type BandwidthConfig struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
}

// RateLimitConfig struct
type RateLimitConfig struct {
	MaxConnectionsFromIP int              `json:"max_connections_from_ip"`
	RPS                  *RPSConfig       `json:"rps"`
	Bandwidth            *BandwidthConfig `json:"bandwidth"`
}

// CountLimit struct
type CountLimit struct {
	sync.Mutex
	m        map[string]int
	maxCount int
}

// Inc method increment limit count
func (c *CountLimit) Inc(ip string) bool {
	c.Lock()
	v, _ := c.m[ip]
	v++

	c.m[ip] = v
	c.Unlock()

	return v <= c.maxCount
}

// Decr method decrement limit count
func (c *CountLimit) Decr(ip string) {
	c.Lock()
	v, _ := c.m[ip]
	if v > 0 {
		v--
	}

	c.m[ip] = v
	c.Unlock()
}

// NewCountLimt func returns CountLimit pointer
func NewCountLimt(maxCount int) *CountLimit {
	return &CountLimit{
		m:        map[string]int{},
		maxCount: maxCount,
	}
}

// RPSLimit struct
type RPSLimit struct {
	sync.Mutex
	count     int
	rps       int
	updatedAt *time.Time
}

// Inc method
func (r *RPSLimit) Inc() bool {
	now := time.Now()
	r.Lock()
	defer r.Unlock()

	if r.updatedAt == nil || r.updatedAt.Add(time.Second).Before(now) {
		r.count = 1
		r.updatedAt = &now
	} else {
		r.count++
	}

	return r.count <= r.rps
}

// NewRPSLimit func returns RPSLimit pointer
func NewRPSLimit(rps int) *RPSLimit {
	return &RPSLimit{
		rps: rps,
	}
}

// BandwidthLimit struct
type BandwidthLimit struct {
	sync.Mutex
	m        map[string]int64
	maxCount int64
	lastDate string
}

// Inc method increment limit count
func (b *BandwidthLimit) Inc(ip string, bytesCount int64) bool {
	now := time.Now().Format("2006-01-02")

	b.Lock()
	defer b.Unlock()

	if b.lastDate != now {
		b.lastDate = now
		b.m = map[string]int64{
			ip: bytesCount,
		}

		return bytesCount <= b.maxCount
	}

	v, _ := b.m[ip]
	v += bytesCount
	b.m[ip] = v

	return v <= b.maxCount
}

// NewBandwidthLimit func returns BandwidthLimit pointer
func NewBandwidthLimit(maxCount int64) *BandwidthLimit {
	return &BandwidthLimit{
		m:        map[string]int64{},
		maxCount: maxCount,
		lastDate: time.Now().Format("2006-01-02"),
	}
}

// RateLimit struct
type RateLimit struct {
	Config            *RateLimitConfig
	maxConnection     *CountLimit
	bandwidthDownload *BandwidthLimit
	bandwidthUpload   *BandwidthLimit
	rpsDownload       *RPSLimit
	rpsUpload         *RPSLimit
	rpsRemove         *RPSLimit
}

// AddConnection method
func (r *RateLimit) AddConnection(ip string) bool {
	if r.maxConnection == nil {
		return true
	}

	return r.maxConnection.Inc(ip)
}

// RemoveConnection method
func (r *RateLimit) RemoveConnection(ip string) {
	if r.maxConnection == nil {
		return
	}

	r.maxConnection.Decr(ip)
}

// CheckBandwidth method
func (r *RateLimit) CheckBandwidth(action, ip string, bytesCount int64) bool {
	switch action {
	case "upload":
		if r.bandwidthUpload == nil {
			return true
		}

		return r.bandwidthUpload.Inc(ip, bytesCount)
	case "download":
		if r.bandwidthDownload == nil {
			return true
		}

		return r.bandwidthDownload.Inc(ip, bytesCount)
	}

	// return true for other actions
	return true
}

// CheckRPS method
func (r *RateLimit) CheckRPS(action string) bool {
	switch action {
	case "upload":
		if r.rpsUpload == nil {
			return true
		}

		return r.rpsUpload.Inc()
	case "download":
		if r.rpsDownload == nil {
			return true
		}

		return r.rpsDownload.Inc()
	case "remove":
		if r.rpsRemove == nil {
			return true
		}

		return r.rpsRemove.Inc()
	}

	// return true for other actions
	return true
}

// NewRateLimit func returns RateLimit pointer
func NewRateLimit(cfg *RateLimitConfig) *RateLimit {
	r := RateLimit{
		Config: cfg,
	}

	if cfg.MaxConnectionsFromIP > 0 {
		r.maxConnection = NewCountLimt(cfg.MaxConnectionsFromIP)
	}

	if cfg.Bandwidth != nil {
		if cfg.Bandwidth.Download > 0 {
			r.bandwidthDownload = NewBandwidthLimit(cfg.Bandwidth.Download)
		}

		if cfg.Bandwidth.Upload > 0 {
			r.bandwidthUpload = NewBandwidthLimit(cfg.Bandwidth.Upload)
		}
	}

	if cfg.RPS != nil {
		if cfg.RPS.Upload > 0 {
			r.rpsUpload = NewRPSLimit(cfg.RPS.Upload)
		}

		if cfg.RPS.Download > 0 {
			r.rpsDownload = NewRPSLimit(cfg.RPS.Download)
		}

		if cfg.RPS.Remove > 0 {
			r.rpsRemove = NewRPSLimit(cfg.RPS.Remove)
		}
	}

	return &r
}
