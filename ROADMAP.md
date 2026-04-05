# grove — Development Milestones

---

## Milestone 0: Repository Setup

**Goal:** A clean, professional Go repository that is ready for development before a single line of library code is written.

**Tasks:**

- Initialize the repo: `git init`, `go mod init github.com/ARJ2211/grove`
- Set the minimum Go version to 1.21 in `go.mod` (generics stable, `slog` available, `errors.Join` available)
- Create the directory structure: `grove/`, `internal/`, `examples/`, `benchmarks/`
- Write `README.md` with the one-paragraph pitch, the core promise, and a placeholder for the API
- Write `CONTRIBUTING.md` with code style rules and the test coverage requirement
- Set up `golangci-lint` with a `.golangci.yml` config covering `errcheck`, `govet`, `staticcheck`, `gosimple`, `unused`
- Set up GitHub Actions CI: lint on every push, test on every push, test with `-race` flag on every push
- Write a `Makefile` with targets: `make test`, `make lint`, `make bench`, `make coverage`
- Add `LICENSE` (MIT)

**Done when:** `go test ./...` passes (no tests yet, just no compile errors), CI is green, lint is clean.

---

## Milestone 1: Error Types

**Goal:** Define the two error types that every other piece of the library depends on. Nothing else gets written until these are solid.

**Files:** `errors.go`

**Tasks:**

`PanicError` — wraps a panic that was caught by grove:

- Holds the original panic value as `interface{}`
- Holds the full stack trace as a `string` captured at the moment of recovery
- Implements `error` interface with a message that includes both the value and the trace
- Has an `Unwrap() error` method for cases where the panic value was itself an error

`MultiError` — collects multiple errors from concurrent tasks:

- Holds a `[]error` slice internally
- Implements `error` interface, formats as a numbered list of all errors
- Implements `Unwrap() []error` so `errors.Is` and `errors.As` work through it
- Has a `Join(errs ...error) error` constructor that returns nil if all errors are nil, a single error if only one is non-nil, and a `MultiError` only if two or more are non-nil
- Follows the same contract as `errors.Join` from Go 1.20 stdlib but with better formatting

**Tests:** 100% coverage. Test every combination: all nil, one non-nil, multiple non-nil, `errors.Is` through a `MultiError`, `errors.As` through a `PanicError`, stack trace is captured correctly, formatting is readable.

**Done when:** `go test -cover ./... | grep errors.go` shows 100.0%.

---

## Milestone 2: Internal Goroutine Wrapper

**Goal:** A panic-safe goroutine launcher that every public API will use internally. This is the lowest-level building block.

**Files:** `internal/goroutine.go`

**Tasks:**

Write a function `Run(name string, fn func() error, errCh chan<- error)` that:

- Launches `fn` in a new goroutine
- Wraps the goroutine body in `defer func() { if r := recover(); r != nil { ... } }()`
- If `fn` panics, captures the panic value and the stack trace via `runtime/debug.Stack()`
- Constructs a `PanicError` from the captured value and stack
- Sends either the function's returned error or the `PanicError` to `errCh`
- If `fn` returns nil, sends nil to `errCh`
- The goroutine's name is embedded in any error message for debuggability

Write a helper `CapturePanic(fn func() (err error))` that wraps a function call inline (not as a goroutine) and converts any panic to a `PanicError`. This will be useful in tests.

**Tests:** Test that a panicking function produces a `PanicError` with the right message, that the stack trace is non-empty, that a normal error passes through unchanged, that nil passes through unchanged, that a panic with a non-error value (e.g. a string) is handled correctly, that `errors.As` can unwrap a `PanicError` from what comes out of `errCh`.

**Done when:** 100% coverage on `internal/goroutine.go`. The package is internal so no public API surface to worry about yet.

---

## Milestone 3: Core Runtime — grove.Run and grove.Go

**Goal:** The heart of the library. `grove.Run` and `grove.Go` working correctly with full test coverage. This is the thing that replaces `errgroup`.

**Files:** `grove.go`

**Tasks:**

Define the `Grove` struct:

```go
type Grove struct {
    ctx    context.Context
    cancel context.CancelCauseFunc
    wg     sync.WaitGroup
    mu     sync.Mutex
    errs   []error
}
```

Implement `Run(ctx context.Context, fn func(*Grove) error) error`:

- Creates a derived context with `context.WithCancelCause`
- Constructs a `Grove` with that context
- Calls `fn(g)` synchronously — this is where the user registers tasks with `g.Go(...)`
- After `fn` returns, calls `g.wg.Wait()` to block until all goroutines finish
- Collects all errors via `MultiError.Join`
- Returns the joined error

Implement `(g *Grove) Go(name string, fn func(ctx context.Context) error)`:

- Increments `g.wg` before launching (important: must happen before the goroutine starts to avoid a race with `Wait`)
- Launches the goroutine using `internal.Run`
- In the goroutine, after the function completes, appends any error to `g.errs` under `g.mu`, calls `g.wg.Done()`, and if the error is non-nil calls `g.cancel(err)` to signal all sibling goroutines
- The context passed to `fn` is `g.ctx` — if any sibling fails, this context is cancelled

Implement `(g *Grove) Context() context.Context` — lets users access the grove's context if needed.

**Tests:**

- Happy path: two goroutines both succeed, `Run` returns nil
- One goroutine fails: `Run` returns that error, the other goroutine's context is cancelled
- Both goroutines fail: `Run` returns a `MultiError` containing both errors
- A goroutine panics: `Run` returns a `PanicError`, server does not crash
- Parent context is cancelled before goroutines finish: goroutines receive cancellation
- `g.Go` called after `fn` returns: must panic with a clear message (grove is closed)
- Zero goroutines: `Run` returns nil immediately
- Nested `grove.Run` inside a goroutine: works correctly, inner grove is scoped to the inner goroutine

**Race detector:** All tests must pass with `go test -race`. This is non-negotiable for a concurrency library.

**Done when:** 100% coverage, all tests pass with `-race`.

---

## Milestone 4: Integration Tests and Examples

**Goal:** Real-world usage patterns work correctly end-to-end. The examples from the proposal all compile and run.

**Files:** `examples/fanout/main.go`, `examples/checkout/main.go`, `grove_test.go` (integration tests)

**Tasks:**

Write the fan-out example:

- Simulates three HTTP service calls using `time.Sleep` and fake clients
- One of the services fails with a configurable probability
- Demonstrates that all errors are collected and that cancellation propagates

Write the checkout example:

- Demonstrates two-scope usage: a server grove and a request grove
- Shows that background tasks (email, analytics) outlive the request handler
- Shows that a panic in a background task does not crash the server

Write integration tests that cover scenarios no unit test can capture:

- Goroutine leak detection: after `Run` returns, `runtime.NumGoroutine()` should equal what it was before `Run` was called
- Correct ordering: `Run` must not return before the last goroutine exits, verified with a time-based test
- Error cancellation timing: after one task fails, siblings receive cancellation within a measurable window
- High concurrency: launch 10,000 goroutines in one `grove.Run`, all succeed, verify zero leaks

**Done when:** All examples compile and produce correct output. Integration tests pass with `-race`.

---

## Milestone 5: grove.Collect[T] — Typed Generic Results

**Goal:** The most novel part of the library. Tasks return typed values, no closure gymnastics required.

**Files:** `collect.go`

**Tasks:**

Define `TypedGrove[T any]` — a grove variant where each task returns a `(T, error)` pair:

```go
type TypedGrove[T any] struct {
    grove   *Grove
    mu      sync.Mutex
    results []T
}
```

Implement `Collect[T any](ctx context.Context, fn func(*TypedGrove[T]) error) ([]T, error)`:

- Creates an inner `Grove` and a `TypedGrove[T]` wrapping it
- Calls `fn` to let the user register tasks
- Waits for all tasks to complete
- Returns all collected `T` values and any errors

Implement `(g *TypedGrove[T]) Submit(name string, fn func(ctx context.Context) (T, error))`:

- Delegates to the inner `Grove.Go`
- On success, appends the returned `T` to `g.results` under `g.mu`
- On failure, the error propagates through the inner grove's error collection

Implement `First[T any](ctx context.Context, fn func(*TypedGrove[T]) error) (T, error)`:

- Same structure as `Collect` but cancels all remaining tasks the moment any one succeeds
- Returns only the first successful result
- If all tasks fail, returns the zero value of T and a `MultiError`
- This is the Happy Eyeballs pattern

Implement `Race[T any](ctx context.Context, fn func(*TypedGrove[T]) error) (T, error)`:

- Returns the result of whichever task completes first, success or failure
- Cancels all remaining tasks immediately after

**Tests:**

- `Collect` with all tasks succeeding: results slice has correct length and values
- `Collect` with one task failing: returns partial results and the error
- `Collect` preserves all results even when some tasks fail (configurable behavior)
- `First` returns the first success and cancels the rest: verify via timing that losers are cancelled
- `First` with all failing: returns MultiError
- `Race` returns fastest result regardless of success/failure
- All tests pass with `-race`

**Done when:** 100% coverage on `collect.go`, all tests pass with `-race`.

---

## Milestone 6: Cancel Scopes

**Goal:** Per-task timeouts and deadlines independent of the parent context.

**Files:** `scope.go`

**Tasks:**

Define `Scope` — a wrapper that adds deadline semantics to a single goroutine registration:

```go
type Scope struct {
    grove    *Grove
    timeout  time.Duration
    deadline time.Time
}
```

Implement `(g *Grove) WithTimeout(d time.Duration) *Scope`

Implement `(g *Grove) WithDeadline(t time.Time) *Scope`

Implement `(s *Scope) Go(name string, fn func(ctx context.Context) error)`:

- Creates a derived context from the grove's context with the scope's timeout/deadline applied
- Launches the goroutine with that scoped context
- When the scoped context expires, `fn` receives cancellation but the parent grove is not cancelled
- If the task returns an error (including context deadline exceeded), that error propagates normally through the grove

The key difference from just using `context.WithTimeout` inside the task: grove knows about the scope and can surface it in debug tooling.

**Tests:**

- Task with 100ms timeout completes in 50ms: succeeds normally
- Task with 100ms timeout takes 200ms: receives cancellation at 100ms, returns `context.DeadlineExceeded`
- Scoped task failing does not cancel sibling tasks (unlike a task failure in the main grove)
- Two tasks with different timeouts: each times out independently
- Parent context cancelled before scope timeout: task receives parent cancellation
- All tests pass with `-race`

**Done when:** 100% coverage on `scope.go`, all tests pass with `-race`.

---

## Milestone 7: Supervision

**Goal:** Long-running goroutines that automatically restart on failure.

**Files:** `supervise.go`

**Tasks:**

Define the `Strategy` type and the three built-in strategies:

```go
type Strategy int

const (
    RestartOnFailure Strategy = iota
    OneForOne
    OneForAll
)
```

Implement `Supervise(ctx context.Context, strategy Strategy, fn func(*Grove) error) error`:

- Opens a grove and runs `fn` to register tasks
- For `RestartOnFailure`: when any task exits with an error or panic, restart that task alone. Loop until `ctx` is cancelled.
- For `OneForOne`: same as `RestartOnFailure` — restart only the failed task
- For `OneForAll`: when any task exits with an error, cancel all other tasks and restart all of them
- In all strategies, a clean exit (nil error) from a task means that task is done and should not be restarted
- Tracks restart counts per task
- Implements exponential backoff between restarts (start at 10ms, max at 30s, jitter applied)
- `Supervise` only returns when `ctx` is cancelled or all tasks have cleanly exited

Define `SuperviseOption` for configurable backoff, max restarts, restart hook:

```go
grove.Supervise(ctx, grove.RestartOnFailure,
    grove.WithMaxRestarts(5),
    grove.WithBackoff(grove.ExponentialBackoff(10*time.Millisecond, 30*time.Second)),
    fn,
)
```

**Tests:**

- Task that fails three times then succeeds: restarts correctly, eventually stays up
- Task that always fails: respects max restarts limit
- `OneForAll`: one task failing causes all to restart, verified by counting restarts
- Context cancellation stops supervision cleanly
- Backoff is applied between restarts: verified with timing
- A task that panics is treated the same as a task that returns an error
- Concurrent supervision of many tasks: no races
- All tests pass with `-race`

**Done when:** 100% coverage on `supervise.go`, all tests pass with `-race`.

---

## Milestone 8: Debug Tooling and Leak Detector

**Goal:** A goroutine tree for debugging and a leak detector for tests.

**Files:** `debug.go`, `testing.go`

**Tasks:**

`debug.go`:

- When `GROVE_DEBUG=1` env var is set, grove maintains an in-memory tree of all active goroutines
- Each node in the tree holds: task name, start time, parent grove ID, current status (running/done/failed)
- `grove.Tree() string` returns a formatted tree string, useful for logging
- `grove.HTTPHandler() http.Handler` returns an HTTP handler that serves the tree as JSON at `/debug/grove`
- The tree is maintained with minimal overhead when debug mode is off (zero-cost disabled path using an interface or build tag)

`testing.go`:

- `grove.DetectLeaks(t testing.TB)` takes a snapshot of `runtime.NumGoroutine()` at call time and registers a cleanup function via `t.Cleanup`
- In the cleanup function, waits up to 100ms for goroutines to settle, then compares the count
- If any grove-owned goroutines are still running, fails the test with a detailed message listing which goroutine names are still alive
- Works correctly with parallel tests

**Tests:**

- Debug mode off: `Tree()` returns empty string, no overhead
- Debug mode on: `Tree()` shows all running tasks, updates as tasks complete
- `DetectLeaks` passes when grove is used correctly
- `DetectLeaks` fails when a goroutine is manually leaked (using a raw `go func()` for comparison)
- `HTTPHandler` returns valid JSON
- All tests pass with `-race`

**Done when:** 100% coverage, all tests pass with `-race`.

---

## Milestone 9: Benchmarks and Performance Validation

**Goal:** Confirm that grove's overhead is within 5% of raw `errgroup` for the common fan-out case.

**Files:** `benchmarks/bench_test.go`

**Tasks:**

Write benchmarks for:

- `BenchmarkGrove_FanOut_N` — N goroutines doing no-op work via `grove.Run`
- `BenchmarkErrgroup_FanOut_N` — same with `errgroup`
- `BenchmarkRawGoroutines_FanOut_N` — same with raw goroutines + WaitGroup
- `BenchmarkGrove_Collect_N` — N typed results via `grove.Collect[int]`
- `BenchmarkGrove_WithPanic` — grove recovering a panic vs errgroup not recovering it
- `BenchmarkSupervise_Restart` — restart latency under `RestartOnFailure`

Run each at N = 10, 100, 1000, 10000.

Profile allocations per operation — grove should not allocate significantly more than errgroup for the common case.

Document results in `benchmarks/RESULTS.md` with the machine spec, Go version, and comparison table.

**Done when:** Grove fan-out is within 5% of errgroup allocations and time per operation at N=100 (the most common real-world size). Results are documented.

---

## Milestone 10: Final Polish and v0.1.0 Release

**Goal:** Everything is documented, examples are complete, the README is the best documentation of structured concurrency in the Go ecosystem.

**Tasks:**

Documentation:

- Every exported type, function, and method has a GoDoc comment explaining what it does, when to use it, and what happens on error
- `README.md` has: the one-liner pitch, the five problems with minimal code snippets, the API surface with examples, the comparison table against errgroup and conc, installation instructions, and links to the proposal document
- `CHANGELOG.md` initialized with v0.1.0 entry

Examples — all four complete and runnable:

- `examples/fanout/` — product page with four services
- `examples/supervision/` — Kafka consumer that restarts on failure
- `examples/happyeyeballs/` — TCP connection racing with `grove.First[net.Conn]`
- `examples/checkout/` — two-scope fire-and-forget

Release checklist:

- `go vet ./...` clean
- `golangci-lint run` clean
- `go test -race -count=3 ./...` passes (run three times to catch flaky tests)
- `go test -cover ./...` shows at least 90% overall (each individual file should be 100%)
- All examples run without error
- Tag `v0.1.0` and push to GitHub
- Write a brief announcement post for the Go subreddit and the Gophers Slack `#showandtell` channel

**Done when:** `v0.1.0` is tagged, README is complete, all examples work, CI is green.

---

## Summary

| Milestone | What gets built                     | Key constraint                             |
| --------- | ----------------------------------- | ------------------------------------------ |
| 0         | Repo, CI, tooling                   | Must pass lint before writing library code |
| 1         | `MultiError`, `PanicError`          | 100% test coverage                         |
| 2         | Internal goroutine wrapper          | 100% test coverage, `-race` clean          |
| 3         | `grove.Run`, `grove.Go`             | 100% test coverage, `-race` clean          |
| 4         | Integration tests, examples         | Leak detection test must pass              |
| 5         | `Collect[T]`, `First[T]`, `Race[T]` | 100% test coverage, `-race` clean          |
| 6         | Cancel scopes                       | 100% test coverage, `-race` clean          |
| 7         | Supervision                         | 100% test coverage, `-race` clean          |
| 8         | Debug tooling, leak detector        | `-race` clean                              |
| 9         | Benchmarks                          | Within 5% of errgroup at N=100             |
| 10        | Docs, polish, v0.1.0 release        | All examples runnable, README complete     |

The rule across all milestones: no milestone is considered done until the one before it has 100% test coverage and passes `-race`. We never carry forward untested code.
