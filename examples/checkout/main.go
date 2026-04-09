package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ARJ2211/grove"
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
				fmt.Println("server closing...")
				return nil
			}
		case <-ticker.C:
			{
				fmt.Println("server status [OK]")
			}
		case <-ctx.Done():
			{
				fmt.Println("server encounted error...")
				return ctx.Err()
			}
		}
	}
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
		// serverGrove.Go("cache-warmup", func(ctx context.Context) error {
		// 	return warmupCache(false)
		// })

		return nil
	})

	if serverErr != nil {
		fmt.Println("SERVER CRASHED!!!")
	}
}
