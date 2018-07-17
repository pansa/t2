package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	pathPtr := flag.String("cfg", "config.json", "path to config")

	flag.Parse()

	cfg, err := NewConfig(*pathPtr)
	if err != nil {
		log.Fatalf("FATAL\t%s\n", err.Error())
	}

	if cfg.Storage == nil {
		log.Fatalf("FATAL\t%s\n", "STORAGE_IS_NIL")
	}

	storage := NewStorage(cfg.Storage)
	rateLimiter := NewRateLimit(cfg.RateLimit)
	redis := NewRedis(cfg.Redis)

	app := NewApplication(cfg, storage, rateLimiter, redis)

	h := NewHandler(app)

	go func() {
		for range time.Tick(10 * time.Minute) {
			err := app.AutoClean()
			if err != nil {
				// @todo log or send metric
			}
		}
	}()

	log.Println(cfg.Host + ":" + strconv.Itoa(cfg.Port))

	// for simplicity we don't handle https requests
	err = http.ListenAndServe(cfg.Host+":"+strconv.Itoa(cfg.Port), h)
	if err != nil {
		// again for simplicity we don't check config values, just check errors
		log.Fatalf("FATAL\t%s\n", err.Error())
	}

}
