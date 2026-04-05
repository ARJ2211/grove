.PHONY: test lint bench coverage

# Run all tests
test:
	go test ./... -race

# Run linter
lint:
	golangci-lint run

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out ./... -race
	go tool cover -func=coverage.out