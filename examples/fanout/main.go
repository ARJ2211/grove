package main

import (
	"errors"
	"time"
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
	Products    *ProductService
	Prices      *PricingService
	ProdReviews *ReviewsSerivce
	Inventory   *InventoryService
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

	pp.ProdReviews.productReviews = map[Product]Reviews{
		Product("apple"):  appleReviews,
		Product("banana"): bananaReviews,
		Product("orange"): orangeReviews,
		Product("lemon"):  lemonReviews,
	}

	// simulate some work
	time.Sleep(latency)

	if expectError {
		return ErrReviewSerivce
	}
	return nil
}

/* ==============================
		ERROR DEFINITIONS
=============================== */

var (
	ErrProductService = errors.New("product service is down")
	ErrPricingSerivce = errors.New("pricing service is down")
	ErrReviewSerivce  = errors.New("review service is down")
)
