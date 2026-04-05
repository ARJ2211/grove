# Contributing to grove

## Rules

- No untested code
- 100% coverage for new files
- All tests must pass with `-race`
- CI must be green

## Setup

```bash
go mod tidy
make lint
make test
```

## Style

- Follow standard Go (`gofmt`, `go vet`)
- Keep code simple and readable
- Prefer clarity over cleverness

## Errors

- Never ignore errors
- Wrap with context when needed
- Ensure compatibility with `errors.Is` / `errors.As`

## Testing

- Cover all edge cases (nil, multiple errors, panics)
- Always run:
```bash
go test -race ./...
```

## PRs

- Keep them small and focused
- Include tests
- Run:
```bash
make lint && make test && make coverage
```

## Philosophy

- Safety: no panics escape, no leaks  
- Simplicity: small, predictable API  
- Correctness: concurrency should just work
