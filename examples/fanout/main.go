package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/ARJ2211/grove"
)

// simulate a product service
type Product string
type ProductService struct {
	products []Product
}

// simulate a pricing service
type PricingService struct {
	productPrices map[Product]float32
}

// simulate a reviews service
type Reviews []string
type ReviewsSerivce struct {
	productReviews map[Product]Reviews
}

// simulate an inventory service
type InventoryService struct {
	productInventory map[Product]int
}

// simulate the main product page that uses these 4 services
// product page will collect the info from the services.
type ProductPage struct {
	Products  *ProductService
	Prices    *PricingService
	Reviews   *ReviewsSerivce
	Inventory *InventoryService
}

/* ==============================
		SERVICE DEFINITIONS
=============================== */

// get the details of the product via some mock.
func (prod *ProductService) Get(
	expectError bool,
	pp *ProductPage,
	latency time.Duration,
) error {
	prods := []Product{
		"apple",
		"banana",
		"orange",
		"lemon",
	}

	prod.products = prods
	pp.Products.products = prods

	// simulate doing some work
	time.Sleep(latency)

	if expectError {
		return ErrProductService
	}
	return nil
}

// get the prices of the different products via some mock.
func (price *PricingService) Get(
	expectError bool,
	pp *ProductPage,
	latency time.Duration,
) error {
	prices := map[Product]float32{
		Product("apple"):  24.3,
		Product("banana"): 14.4,
		Product("orange"): 32.6,
		Product("lemon"):  12.2,
	}

	price.productPrices = prices
	pp.Prices.productPrices = prices

	// simulate doing some work
	time.Sleep(latency)

	if expectError {
		return ErrPricingSerivce
	}
	return nil
}

// get the reviews for the products via some mock
func (rs *ReviewsSerivce) Get(
	expectError bool,
	pp *ProductPage,
	latency time.Duration,
) error {
	appleReviews := Reviews([]string{
		"Very sweet apples",
		"Apples were crunchy",
		"Extremely juicy",
	})
	bananaReviews := Reviews([]string{
		"Yellow bananas",
		"They were ripe",
	})
	orangeReviews := Reviews([]string{
		"Good for making juices",
		"They were bitter",
		"My son liked them a lot",
	})
	lemonReviews := Reviews([]string{
		"Very sour",
		"I did not like them",
		"Do not recommend",
		"Not enough juice",
	})

	prodRevs := map[Product]Reviews{
		Product("apple"):  appleReviews,
		Product("banana"): bananaReviews,
		Product("orange"): orangeReviews,
		Product("lemon"):  lemonReviews,
	}

	rs.productReviews = prodRevs
	pp.Reviews.productReviews = prodRevs

	// simulate some work
	time.Sleep(latency)

	if expectError {
		return ErrReviewSerivce
	}
	return nil
}

// get the inventory for the products via some mock.
func (inv *InventoryService) Get(
	expectError bool,
	pp *ProductPage,
	latency time.Duration,
) error {
	inventory := map[Product]int{
		Product("apple"):  5,
		Product("orange"): 14,
		Product("banana"): 3,
		Product("lemon"):  7,
	}

	inv.productInventory = inventory
	pp.Inventory.productInventory = inventory

	// simulate some work
	time.Sleep(latency)

	if expectError {
		return ErrInventorySerivce
	}
	return nil
}

/* ==============================
		MAIN ENTRY POINT
=============================== */

// main function where our http mocks will be created
// and ran under a grove
func main() {
	fmt.Print("\033[H\033[2J")
	var t0 time.Time

	prodService := &ProductService{}
	prodPrices := &PricingService{}
	prodReviews := &ReviewsSerivce{}
	prodInventory := &InventoryService{}

	pp := &ProductPage{
		Products:  prodService,
		Prices:    prodPrices,
		Inventory: prodInventory,
		Reviews:   prodReviews,
	}

	// first grove with no errors (happy path)
	t0 = time.Now()
	happyCtx := context.Background
	err := grove.Run(happyCtx(), func(g *grove.Grove) error {
		g.Go("fetch-prods", func(ctx context.Context) error {
			return prodService.Get(false, pp, d())
		})

		g.Go("fetch-prices", func(ctx context.Context) error {
			return prodPrices.Get(false, pp, d())
		})

		g.Go("fetch-reviews", func(ctx context.Context) error {
			return prodReviews.Get(false, pp, d())
		})

		g.Go("fetch-inv", func(ctx context.Context) error {
			return prodInventory.Get(false, pp, d())
		})

		return nil
	})

	if err != nil {
		fmt.Printf("expected nill err in happy path, got: %v", err)
		os.Exit(1)
	}
	printProductPage(pp, time.Since(t0))

	// second grove with one error (single error path)
	t0 = time.Now()
	oneErrorCtx := context.Background()

	err = grove.Run(oneErrorCtx, func(g *grove.Grove) error {
		g.Go("fetch-prods", func(ctx context.Context) error {
			return prodService.Get(false, pp, d())
		})

		// expect fetch prices to throw an error.
		g.Go("fetch-prices", func(ctx context.Context) error {
			return prodPrices.Get(true, pp, d())
		})

		g.Go("fetch-reviews", func(ctx context.Context) error {
			return prodReviews.Get(false, pp, d())
		})

		g.Go("fetch-inv", func(ctx context.Context) error {
			return prodInventory.Get(false, pp, d())
		})

		return nil
	})

	if err == nil {
		fmt.Println("expected one error, got nil")
		os.Exit(1)
	} else if errors.Is(err, ErrPricingSerivce) {
		fmt.Println("[CASE 2] One error path complete!")
		fmt.Printf("Time to complete -> %.2f sec\n", time.Since(t0).Seconds())
		fmt.Printf("error: %v", err)
	} else {
		fmt.Println("One error path complete!")
		fmt.Printf("Time to complete -> %.2f sec\n", time.Since(t0).Seconds())
		fmt.Printf("Caught unknown error, got: %v", err)
		os.Exit(1)
	}

	// third grove with multi error (multiple error path)
	t0 = time.Now()
	multiErrorCtx := context.Background()

	err = grove.Run(multiErrorCtx, func(g *grove.Grove) error {
		g.Go("fetch-prods", func(ctx context.Context) error {
			return prodService.Get(false, pp, d())
		})

		// expect fetch prices to throw an error.
		g.Go("fetch-prices", func(ctx context.Context) error {
			return prodPrices.Get(true, pp, d())
		})

		// expect fetch reviews to throw an error.
		g.Go("fetch-reviews", func(ctx context.Context) error {
			return prodReviews.Get(true, pp, d())
		})

		g.Go("fetch-inv", func(ctx context.Context) error {
			return prodInventory.Get(false, pp, d())
		})

		return nil
	})

	var me grove.MultiError
	if err == nil {
		fmt.Println("expected one error, got nil")
		os.Exit(1)
	} else if errors.Is(err, ErrPricingSerivce) &&
		errors.Is(err, ErrReviewSerivce) &&
		errors.As(err, &me) {
		fmt.Println("\n\n[CASE 3] Multi error path complete!")
		fmt.Printf("Time to complete -> %.2f sec\n", time.Since(t0).Seconds())
		fmt.Printf("Successfully caught multierror %v", me)
	} else {
		fmt.Println("Multi error path complete!")
		fmt.Printf("Time to complete -> %.2f sec\n", time.Since(t0).Seconds())
		fmt.Printf("Caught unknown error, got: %v", err)
		os.Exit(1)
	}

}

// function to get duration
// where durationE[500, 1000]
func d() time.Duration {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	min := 500
	max := 1000

	n := rand.Intn(max-min+1) + min
	t := time.Duration(n) * time.Millisecond
	return t
}

// function to print the product page
func printProductPage(pp *ProductPage, elapsed time.Duration) {
	fmt.Println("\n\n[CASE 1] Happy path complete!")
	fmt.Printf("Time to complete -> %.2f sec\n", elapsed.Seconds())
	fmt.Println("Product Page Details:")

	for _, product := range pp.Products.products {
		fmt.Printf("• %s\n", product)

		// price
		if price, ok := pp.Prices.productPrices[product]; ok {
			fmt.Printf("   Price: $%.2f\n", price)
		}

		// inventory
		if inv, ok := pp.Inventory.productInventory[product]; ok {
			fmt.Printf("   Stock: %d units\n", inv)
		}

		// reviews
		if reviews, ok := pp.Reviews.productReviews[product]; ok {
			fmt.Println("   Reviews:")
			for i, r := range reviews {
				fmt.Printf("     %d. %s\n", i+1, r)
			}
		}

		fmt.Println()
	}
}

/* ==============================
		ERROR DEFINITIONS
=============================== */

var (
	ErrProductService   = errors.New("product service is down")
	ErrPricingSerivce   = errors.New("pricing service is down")
	ErrReviewSerivce    = errors.New("review service is down")
	ErrInventorySerivce = errors.New("inventory service is down")
)
