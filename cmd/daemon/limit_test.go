package main

import (
	"testing"
	"time"
)

func TestNewCountLimit(t *testing.T) {
	cases := []struct {
		maxCount int
	}{
		{
			maxCount: 1024,
		},
		{
			maxCount: 2048,
		},
	}

	for _, tc := range cases {
		v := NewCountLimt(tc.maxCount)

		if v.maxCount != tc.maxCount {
			t.Errorf("Max count must be %d but got %d\n", tc.maxCount, v.maxCount)
		}

		if v.m == nil {
			t.Error("Map must not be nil")
		}
	}
}

func TestCountLimitInc(t *testing.T) {
	v := NewCountLimt(2)

	cases := []struct {
		ip string
		ok bool
	}{
		{
			ip: "127.0.0.1",
			ok: true,
		},
		{
			ip: "127.0.0.1",
			ok: true,
		},
		{
			ip: "127.0.0.1",
			ok: false,
		},
		{
			ip: "127.0.0.2",
			ok: true,
		},
		{
			ip: "127.0.0.2",
			ok: true,
		},
	}

	for _, tc := range cases {
		ok := v.Inc(tc.ip)
		if tc.ok != ok {
			t.Errorf("Ok must be %t but got %t\n", tc.ok, ok)
		}
	}
}

func TestCountLimitDecr(t *testing.T) {
	v := NewCountLimt(2)

	cases := []struct {
		ip    string
		start int
		res   int
	}{
		{
			start: 3,
			ip:    "127.0.0.1",
			res:   2,
		},
		{
			ip:  "127.0.0.1",
			res: 1,
		},
		{
			ip:  "127.0.0.1",
			res: 0,
		},
		{
			start: 1,
			ip:    "127.0.0.2",
			res:   0,
		},
		{
			ip:  "127.0.0.2",
			res: 0,
		},
	}

	for _, tc := range cases {
		if tc.start > 0 {
			v.m[tc.ip] = tc.start
		}

		v.Decr(tc.ip)
		if tc.res != v.m[tc.ip] {
			t.Errorf("Res must be %d but got %d\n", tc.res, v.m[tc.ip])
		}
	}
}

func TestNewBandwidthLimit(t *testing.T) {
	cases := []struct {
		maxCount int64
	}{
		{
			maxCount: 1024,
		},
		{
			maxCount: 2048,
		},
	}

	for _, tc := range cases {
		v := NewBandwidthLimit(tc.maxCount)
		now := time.Now().Format("2006-01-02")

		if v.maxCount != tc.maxCount {
			t.Errorf("Max count must be %d but got %d\n", tc.maxCount, v.maxCount)
		}

		if now != v.lastDate {
			t.Errorf("Last date must be %s but got %s\n", now, v.lastDate)
		}

		if v.m == nil {
			t.Error("Map must not be nil")
		}
	}
}

func TestBandwidthInc(t *testing.T) {
	v := NewBandwidthLimit(1024)

	cases := []struct {
		ip         string
		bytesCount int64
		ok         bool
	}{
		{
			ip:         "127.0.0.1",
			bytesCount: 800,
			ok:         true,
		},
		{
			ip:         "127.0.0.1",
			bytesCount: 800,
			ok:         false,
		},
		{
			ip:         "127.0.0.2",
			bytesCount: 1024,
			ok:         true,
		},
		{
			ip:         "127.0.0.3",
			bytesCount: 2048,
			ok:         false,
		},
	}

	for i := 0; i < 2; i++ {
		for index, tc := range cases {
			if i == 1 && index == 0 {
				v.lastDate = time.Now().Add(-24 * time.Hour).Format("2006-01-02")
			}

			ok := v.Inc(tc.ip, tc.bytesCount)
			if tc.ok != ok {
				t.Errorf("Ok must be %t but got %t\n", tc.ok, ok)
			}
		}
	}

}

func TestNewRPSLimit(t *testing.T) {
	cases := []struct {
		rps int
	}{
		{
			rps: 1024,
		},
		{
			rps: 2048,
		},
	}

	for _, tc := range cases {
		v := NewRPSLimit(tc.rps)

		if v.rps != tc.rps {
			t.Errorf("RPS must be %d but got %d\n", tc.rps, v.rps)
		}
	}
}

func TestRPSLimitInc(t *testing.T) {
	v := NewRPSLimit(3)

	cases := []struct {
		ok bool
	}{
		{
			ok: true,
		},
		{
			ok: true,
		},
		{
			ok: true,
		},
		{
			ok: false,
		},
		{
			ok: false,
		},
	}

	for index, tc := range cases {
		ok := v.Inc()

		if tc.ok != ok {
			t.Errorf("Ok must be %t but got %t (%d request)\n", tc.ok, ok, index)
		}
	}
}

func TestNewRateLimit(t *testing.T) {
	cases := []struct {
		cfg RateLimitConfig
	}{
		{
			cfg: RateLimitConfig{
				MaxConnectionsFromIP: 3,
			},
		},
		{
			cfg: RateLimitConfig{
				MaxConnectionsFromIP: 3,
				Bandwidth: &BandwidthConfig{
					Download: 1000000,
					Upload:   1000000,
				},
				RPS: &RPSConfig{
					Download: 10,
					Upload:   10,
					Remove:   10,
				},
			},
		},
	}

	for _, tc := range cases {
		v := NewRateLimit(&tc.cfg)

		if v == nil {
			t.Error("Res must not be nil")
		}
	}
}

func TestRateLimitAddConnection(t *testing.T) {
	cases := []struct {
		cfg *RateLimitConfig
		ip  string
		ok  bool
	}{
		{
			cfg: &RateLimitConfig{},
			ip:  "127.0.0.1",
			ok:  true,
		},
		{
			cfg: &RateLimitConfig{
				MaxConnectionsFromIP: 10,
			},
			ip: "127.0.0.1",
			ok: true,
		},
	}

	for _, tc := range cases {
		v := NewRateLimit(tc.cfg)

		ok := v.AddConnection(tc.ip)

		if tc.ok != ok {
			t.Errorf("Ok must be %t but got %t\n", tc.ok, ok)
		}
	}
}

func TestRateLimitRemoveConnection(t *testing.T) {
	cases := []struct {
		cfg *RateLimitConfig
		ip  string
		ok  bool
	}{
		{
			cfg: &RateLimitConfig{},
			ip:  "127.0.0.1",
			ok:  true,
		},
		{
			cfg: &RateLimitConfig{
				MaxConnectionsFromIP: 10,
			},
			ip: "127.0.0.1",
			ok: true,
		},
	}

	for _, tc := range cases {
		v := NewRateLimit(tc.cfg)

		v.RemoveConnection(tc.ip)
	}
}

func TestRateLimitCheckBandwidth(t *testing.T) {
	cases := []struct {
		cfg        *RateLimitConfig
		ip         string
		action     string
		bytesCount int64
		ok         bool
	}{
		{
			cfg:    &RateLimitConfig{},
			ip:     "127.0.0.1",
			action: "download",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				Bandwidth: &BandwidthConfig{},
			},
			ip:     "127.0.0.1",
			action: "download",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				Bandwidth: &BandwidthConfig{
					Download: 1024,
				},
			},
			bytesCount: 2048,
			ip:         "127.0.0.1",
			action:     "download",
			ok:         false,
		},
		{
			cfg:    &RateLimitConfig{},
			ip:     "127.0.0.1",
			action: "upload",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				Bandwidth: &BandwidthConfig{},
			},
			ip:     "127.0.0.1",
			action: "upload",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				Bandwidth: &BandwidthConfig{
					Upload: 1024,
				},
			},
			bytesCount: 2048,
			ip:         "127.0.0.1",
			action:     "upload",
			ok:         false,
		},
		{
			cfg: &RateLimitConfig{
				Bandwidth: &BandwidthConfig{
					Upload:   1024,
					Download: 1024,
				},
			},
			bytesCount: 2048,
			ip:         "127.0.0.1",
			action:     "unknown",
			ok:         true,
		},
	}

	for _, tc := range cases {
		v := NewRateLimit(tc.cfg)

		ok := v.CheckBandwidth(tc.action, tc.ip, tc.bytesCount)

		if tc.ok != ok {
			t.Errorf("Ok must be %t but got %t\n", tc.ok, ok)
		}
	}
}

func TestRateLimitCheckRPS(t *testing.T) {
	cases := []struct {
		cfg    *RateLimitConfig
		action string
		ok     bool
	}{
		{
			cfg:    &RateLimitConfig{},
			action: "download",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				RPS: &RPSConfig{},
			},
			action: "download",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				RPS: &RPSConfig{
					Download: 10,
				},
			},
			action: "download",
			ok:     true,
		},
		{
			cfg:    &RateLimitConfig{},
			action: "upload",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				RPS: &RPSConfig{},
			},
			action: "upload",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				RPS: &RPSConfig{
					Upload: 10,
				},
			},
			action: "upload",
			ok:     true,
		},
		{
			cfg:    &RateLimitConfig{},
			action: "remove",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				RPS: &RPSConfig{},
			},
			action: "remove",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				RPS: &RPSConfig{
					Remove: 10,
				},
			},
			action: "remove",
			ok:     true,
		},
		{
			cfg: &RateLimitConfig{
				RPS: &RPSConfig{
					Download: 10,
					Upload:   10,
					Remove:   10,
				},
			},
			action: "unknown",
			ok:     true,
		},
	}

	for _, tc := range cases {
		v := NewRateLimit(tc.cfg)

		ok := v.CheckRPS(tc.action)

		if tc.ok != ok {
			t.Errorf("Ok must be %t but got %t\n", tc.ok, ok)
		}
	}
}
