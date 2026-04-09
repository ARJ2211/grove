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
	PANIC_CACHE     = true  // panic the cache warmup after 5 seconds
	FAIL_SAVE_ORDER = false // fail the order saving part done in request context

	ErrOrderNotSaved = errors.New("order not saved")
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
				fmt.Println("[server] server encountered an error. " +
					"Completing all previous goroutines")
				return ctx.Err()
			}
		}
	}
}

// simple simulation of cache warmup for a server.
func warmupCache(expectPanic bool) error {
	fmt.Println("[cache] cache is warming up...")

	if expectPanic {
		// if cache warmup fails, we panic!
		time.Sleep(5 * time.Second)
		e := errors.New("cache warmup failed!")
		panic(e)
	}

	time.Sleep(10 * time.Second)
	fmt.Println("[cache] cache is warmed up.")

	return nil
}

/* ======================
	REQUEST HANDLERS
====================== */

// checkout request simulation.
func checkoutHandler(serverGrove *grove.Grove) error {
	reqCtx := context.Background()

	reqErr := grove.Run(reqCtx, func(g *grove.Grove) error {
		// charge the users card
		g.Go("charge-card", func(ctx context.Context) error {
			simulateWork(200)
			fmt.Println(
				"[r1 charge-card] your card ending with xxxx has been charged",
			)
			return nil
		})

		// save the order in the DB
		g.Go("save-order", func(ctx context.Context) error {
			simulateWork(150)
			if FAIL_SAVE_ORDER {
				fmt.Println(
					"[r1 save-order] your order has NOT been saved",
				)
				return ErrOrderNotSaved
			} else {
				fmt.Println(
					"[r1 save-order] your order has been saved",
				)
				return nil
			}

		})

		return nil
	})

	if reqErr != nil {
		// catch any error that happened with the card
		// and refund the money.
		/*
			if errors.As(r1Err != ErrFaultyTransaction) {
				refund()
			}
		*/
		fmt.Println("[req] error occured - any money will be refunded")
		return reqErr
	}

	// we fire the background job that needs to be done in the servers
	// context since these can be long lived tasks.
	serverGrove.Go("send-email", func(ctx context.Context) error {
		simulateWork(5000)
		fmt.Println("[send-email] email sent to user")
		return nil
	})
	serverGrove.Go("update-inventory", func(ctx context.Context) error {
		// when cache error is set to true, even though the cache fails at
		// t = 5 secs, since we are not checking the context, the goroutine
		// still executes until completion!
		simulateWork(10000)
		fmt.Println("[update-inventory] inventory updated")
		return nil
	})

	return nil
}

func main() {
	var pe grove.PanicError

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
			err := warmupCache(PANIC_CACHE) // <- set true or false to simulate cache crash
			return err
		})

		// inner grove that handles different requests
		// request 1: checkout request
		serverGrove.Go("checkout-req", func(ctx context.Context) error {
			err := checkoutHandler(serverGrove)
			if err != nil {
				return err
			}
			return nil
		})

		return nil
	})

	if errors.As(serverErr, &pe) {
		fmt.Println("\n\nYour server crashed but all " +
			"requests were served before crashing",
		)
	}
	if serverErr != nil {
		fmt.Println("[server] server encounted error: ", serverErr)
	}
}

func simulateWork(d int) {
	dur := d * int(time.Millisecond)
	// fmt.Printf("\n[dur] %d\n", dur)
	time.Sleep(time.Duration(dur))
}
