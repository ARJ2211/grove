package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ARJ2211/grove"
)

// some flags to play around with
var (
	FAILCACHE = false // fail the cache warmup after 5 seconds
)

/* ======================
	BACKGROUND TASKS
====================== */

// simple simulation of health check for a server.
func healthCheck(ctx context.Context, timer *time.Timer, ticker *time.Ticker) error {
	for {
		select {
		case <-timer.C:
			{
				fmt.Println("[server] server closing...")
				return nil
			}
		case <-ticker.C:
			{
				fmt.Println("[server] server status [OK]")
			}
		case <-ctx.Done():
			{
				fmt.Println("[server] server encounted error...")
				return ctx.Err()
			}
		}
	}
}

// simple simulation of cache warmup for a server.
func warmupCache(expectError bool) error {
	fmt.Println("[cache] cache is warming up...")

	if expectError {
		// if cache warmup fails, we panic!
		time.Sleep(5 * time.Second)
		e := errors.New("cache warmup failed!")
		panic(e)
	}

	time.Sleep(10 * time.Second)
	fmt.Println("[cache] cache is warmed up.")

	return nil
}

func main() {
	serverCtx := context.Background()

	timer := time.NewTimer(45 * time.Second)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	fmt.Println("server started...")

	serverErr := grove.Run(serverCtx, func(serverGrove *grove.Grove) error {
		// background task 1: health-check
		serverGrove.Go("health-check", func(ctx context.Context) error {
			return healthCheck(ctx, timer, ticker)
		})

		// background task 2: cache warmup
		serverGrove.Go("cache-warmup", func(ctx context.Context) error {
			err := warmupCache(FAILCACHE) // <- set true or false to simulate cache crash
			return err
		})

		// request: this handles the checkout process
		// checkoutProcess()

		return nil
	})

	if serverErr != nil {
		fmt.Printf("SERVER CRASHED with error: %v", serverErr.Error())
	}
}
