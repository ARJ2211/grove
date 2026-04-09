/*
	This file serves as an example of how Grove can be used in
	a server setting by simulating a user checkout process.

	This file will explain the two-scope pattern where we can
	have nested groves and how they benefit us in a server
	setting where we have long lived tasks and short lived
	tasks. This pattern also explains how a panic or an error
	in one goroutine does not crash the server mid request

	In a traditional setting, when a server handles a request
	like a checkout request, a request handler fires under the
	same context as the request itself. The request handler that
	handles this checkout process may employ some goroutines that
	it may want to do concurrently or in the background. The
	different functions are as below:
		1.	*charge-card*: This will validate the users card
			and then charge the required amount.
		2.	*save-order*: This will save the users order
			in the database.
		3.	*send-email*: This will send a confirmation email
			to the user.
		4.	*update-inventory*: This will update the inventory
			in the database

	The server may also have some of its own background tasks like
	the ones mentioned below:
		1.	*health-check*: This checks the health of the server
			every one second
		2.	*warm-cache*: At a server startup, a server may want to
			warm up its cache for quicker responses.

	------------------------------
	How Grove fits in all of this:
	------------------------------
	When, say for example, the cache warm up service fails (which takes
	5 seconds to complete). The server (without Grove) panics and shuts down, terminating all ongoing requests. By running these background tasks
	in their own Grove, grove will save the panic, but will also allow all
	existing goroutines to finish executing and then gracefully shut down
	the server.

	In this example, we have the two server background tasks running
	in their own grove, that is, the server context. When a request comes
	in to checkout a users item, the checkoutHandler is called. The
	checkout handler opens up its own grove under the context of the request
	as this is a short lived task and executes the charge-card and save-order
	functions concurrently.

	If the grove does not return an error, then it proceeds to do the longer
	lived tasks like sending the user email and updating the inventory.

	If the cache warmup function panics, the server Grove catches that panic
	as a PanicError and waits for all goroutines to complete (in this case a
	user request that came during the cache warmup) thus not crashing the
	server immediately.

	This is the two-scope pattern where:
		1. 	The server is the long lived scope
		2. 	The request (charge-card & save-order) are short lived
			in the scope of the request
		3.	The send-email and update-inventory functions are long
			lived too as they become background tasks that the user
			need not wait for, hence run using the serverGrove.

	There are some flags that one can play around with:
	1. 	PANIC_CACHE = false && FAIL_SAVE_ORDER = false:
		This is the best case scenario where the server will
		run for the entirety of its duration (45 seconds)

	2.	PANIC_CACHE = true && FAIL_SAVE_ORDER = false:
		In this scenario, Grove will save the panic from the
		cache warmup function and gracefully shut down the
		server when ALL the server goroutines (requests) have
		been completed.

	3.	PANIC_CACHE = true && FAIL_SAVE_ORDER = true:
		Here Grove will return a multi-error (grove.MultiError)
		where one can track the errors and the call stack from
		the panic. The background tasks to send the email
		and update the inventory will not run as we call on the
		refund() method while handling the error from the nested
		request grove.

	4.	PANIC_CACHE = false && FAIL_SAVE_ORDER = true:
		Here Grove will start propagating the error up to the server
		but we can catch it in where we call checkoutHandler() and check
		if the error is a ErrOrderNotSaved error and not propagate
		it to the server, not causing a full server shutdown.
*/

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
	PANIC_CACHE     = true // panic the cache warmup after 5 seconds
	FAIL_SAVE_ORDER = true // fail the order saving part done in request context

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
		e := errors.New("cache warmup failed")
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

/* ======================
	RUNNING MOCK SERVER
====================== */

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

			// we dont want to propogate any request error back
			// up to the server.
			if err != nil {
				if errors.Is(err, ErrOrderNotSaved) {
					// handle refund, log it, but don't propagate
					fmt.Println("[req] order not saved, refunding...")
				} else {
					// unknown error, propagate to server
					return err
				}
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

// simple function to simulate doing
// work in milliseconds.
func simulateWork(d int) {
	dur := d * int(time.Millisecond)
	time.Sleep(time.Duration(dur))
}
