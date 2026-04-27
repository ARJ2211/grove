<p align="center">
  <img src="assets/grove_logo.svg" width="550" alt="grove">
</p>

Grove is a Go library that gives goroutines a scope, an owner, and a lifetime. The core promise: when `grove.Run` returns, every goroutine it spawned has finished. No exceptions, no leaks, no goroutines running in the background.

Unstructured goroutines are the concurrency equivalent of the `goto` statement. Just as unstructured jumps make programs impossible to reason about, unstructured goroutines make concurrent Go programs impossible to reason about. Grove enforces structure.

---

## The Problem with errgroup

Go's `errgroup` is the closest thing in the standard library to structured concurrency, and it is genuinely good. But it has four concrete limitations:

1. **First error only.** If three services fail simultaneously, you see one error. The other two disappear silently.
2. **No typed results.** Every task must smuggle its return value out through a closure capture. With generics available since Go 1.18, this is an unnecessary constraint.
3. **No supervision.** For long-running workers, every Go service reinvents its own restart loop from scratch. There is no standard pattern.
4. **No panic recovery.** A panic in any goroutine kills the entire process. `errgroup` does not recover panics.

Grove fixes all four.

---

## Installation

```bash
go get github.com/ARJ2211/grove
```

Requires Go 1.21 or later.

---

## API Overview

| Symbol                  | What it does                                                                      |
| ----------------------- | --------------------------------------------------------------------------------- |
| `grove.Run`             | Runs goroutines under a shared scope, waits for all, returns all errors           |
| `grove.Go`              | Registers a named goroutine inside a `Run` scope                                  |
| `grove.Collect[T]`      | Like `Run`, but each task returns a typed value                                   |
| `grove.First[T]`        | Returns the first successful typed result, cancels the rest                       |
| `grove.Race[T]`         | Returns the first result regardless of success or failure                         |
| `grove.MultiError`      | Holds multiple errors from concurrent tasks, supports `errors.Is` and `errors.As` |
| `grove.PanicError`      | Wraps a recovered panic, includes the full stack trace                            |
| `(*Grove).WithTimeout`  | Creates a cancel scope with a timeout for a single task                           |
| `(*Grove).WithDeadline` | Creates a cancel scope with a deadline for a single task                          |

---

## Core Usage

### grove.Run

`grove.Run` is the replacement for `errgroup`. It runs all registered goroutines, waits for every one to finish, and returns all errors collected across them.

```go
err := grove.Run(ctx, func(g *grove.Grove) error {
    g.Go("fetch-products", func(ctx context.Context) error {
        return productService.Fetch(ctx)
    })
    g.Go("fetch-prices", func(ctx context.Context) error {
        return pricingService.Fetch(ctx)
    })
    g.Go("fetch-inventory", func(ctx context.Context) error {
        return inventoryService.Fetch(ctx)
    })
    return nil
})
```

When any task fails, the grove's context is cancelled, signalling all sibling goroutines. When `Run` returns, every goroutine has exited.

### Collecting All Errors

If multiple tasks fail, `grove.Run` returns a `MultiError` rather than dropping all but the first.

```go
var me grove.MultiError
if errors.As(err, &me) {
    for i, e := range me.Unwrap() {
        fmt.Printf("error %d: %v\n", i+1, e)
    }
}
```

`MultiError` supports `errors.Is` and `errors.As` for traversal through the chain.

### Panic Recovery

A panic in any goroutine is caught, wrapped in a `PanicError` with the full stack trace, and returned as a normal error. The process does not crash.

```go
err := grove.Run(ctx, func(g *grove.Grove) error {
    g.Go("unstable-task", func(ctx context.Context) error {
        panic("something went wrong")
    })
    return nil
})

var pe grove.PanicError
if errors.As(err, &pe) {
    fmt.Println(pe.Error()) // includes panic value and stack trace
}
```

---

## Typed Results with Generics

### grove.Collect[T]

When tasks produce values, `Collect` eliminates the closure-capture pattern. Each task returns a typed value directly.

```go
results, err := grove.Collect[string](ctx, func(tg *grove.TypedGrove[string]) error {
    tg.Submit("service-a", func(ctx context.Context) (string, error) {
        return serviceA.Fetch(ctx)
    })
    tg.Submit("service-b", func(ctx context.Context) (string, error) {
        return serviceB.Fetch(ctx)
    })
    return nil
})

// results is []string containing all successful return values
```

### grove.First[T]

Runs all tasks and returns the first successful result. All remaining tasks are cancelled the moment one succeeds. This is the Happy Eyeballs pattern.

```go
conn, err := grove.First[net.Conn](ctx, func(tg *grove.TypedGrove[net.Conn]) error {
    tg.SubmitFirst("ipv4", func(ctx context.Context) (net.Conn, error) {
        return net.Dial("tcp4", addr)
    })
    tg.SubmitFirst("ipv6", func(ctx context.Context) (net.Conn, error) {
        return net.Dial("tcp6", addr)
    })
    return nil
})
```

If all tasks fail, `First` returns a `MultiError` containing every failure.

### grove.Race[T]

Returns the result of whichever task completes first, whether it succeeded or failed. All remaining tasks are cancelled immediately after.

```go
result, err := grove.Race[Response](ctx, func(tg *grove.TypedGrove[Response]) error {
    tg.SubmitRace("primary", func(ctx context.Context) (Response, error) {
        return primary.Call(ctx)
    })
    tg.SubmitRace("fallback", func(ctx context.Context) (Response, error) {
        return fallback.Call(ctx)
    })
    return nil
})
```

---

## Cancel Scopes

A cancel scope gives a single task its own timeout or deadline, completely independent of the parent grove. When a scoped task times out, its siblings keep running. The parent grove is not cancelled.

This is the key difference from calling `context.WithTimeout` inside a task: in a regular `g.Go` task, any returned error cancels all siblings. In a scoped task, it does not.

### WithTimeout

```go
err := grove.Run(ctx, func(g *grove.Grove) error {
    // This task has 200ms to complete. If it times out, only it fails.
    // The tasks below are not cancelled.
    scope := g.WithTimeout(200 * time.Millisecond)
    scope.Go("optional-analytics", func(ctx context.Context) error {
        return analytics.Send(ctx, payload)
    })

    // These tasks run with the full parent context.
    g.Go("critical-task-a", func(ctx context.Context) error {
        return serviceA.Call(ctx)
    })
    g.Go("critical-task-b", func(ctx context.Context) error {
        return serviceB.Call(ctx)
    })

    return nil
})
```

### WithDeadline

```go
err := grove.Run(ctx, func(g *grove.Grove) error {
    deadline := time.Now().Add(500 * time.Millisecond)
    scope := g.WithDeadline(deadline)
    scope.Go("time-sensitive-task", func(ctx context.Context) error {
        return service.Call(ctx)
    })
    return nil
})
```

### Behaviour Summary

| Scenario                                      | What happens                                            |
| --------------------------------------------- | ------------------------------------------------------- |
| Scoped task completes before timeout          | Returns nil, no effect on siblings                      |
| Scoped task exceeds timeout                   | Returns `context.DeadlineExceeded`, siblings unaffected |
| Parent context cancelled before scope timeout | Scoped task receives `context.Canceled`                 |
| Two scoped tasks with different timeouts      | Each times out independently                            |

---

## Error Types

### MultiError

Returned by `Run` when two or more goroutines fail. Formats as a numbered list and supports full error chain traversal.

```
3 errors occurred:
   [1]: pricing service is down
   [2]: inventory service is down
   [3]: review service is down
```

### PanicError

Returned when a goroutine panics. Contains the original panic value and the full stack trace captured at the moment of recovery. If the panic value was itself an `error`, `errors.As` and `errors.Is` can unwrap through it.

---

## Error Handling Reference

```go
// Check if any specific error is present anywhere in the chain
if errors.Is(err, ErrPricingService) { ... }

// Extract the MultiError to iterate all failures
var me grove.MultiError
if errors.As(err, &me) {
    for _, e := range me.Unwrap() { ... }
}

// Extract a PanicError to read the stack trace
var pe grove.PanicError
if errors.As(err, &pe) {
    log.Println(pe.Error())
}
```

---

## Design Principles

**Every goroutine has an owner.** `grove.Run` owns the goroutines registered inside it. When `Run` returns, ownership is released and every goroutine is guaranteed to have exited.

**All errors are surfaced.** No error is silently dropped. If five tasks fail, you get five errors.

**Panics are errors, not crashes.** Any panic in any grove-owned goroutine is caught, wrapped with its stack trace, and returned as a normal error.

**Cancel scopes do not poison siblings.** A timeout on one task is that task's problem alone. The rest of the grove continues running.

**The context is always honoured.** Every goroutine receives the grove's context. If the parent context is cancelled, all goroutines are notified immediately.

---

## License

Apache2.0. See [LICENSE](./LICENSE).
