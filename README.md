# grove

Grove is a `Go` library that gives goroutines a scope, an owner, and a lifetime. The core promise of grove being that when `grove.Run` returns, every goroutine spawned inside it has finished. No exceptions, no leaks, and no goroutines left running in the background.

Grove is a golang package that can be used by developers to bring structure to their concurrent programming as this is the concurrency equivalent of the goto statement that Dijkstra argued against in 1968 in his paper titled "Edgar Dijkstra: Go To Statement Considered Harmful". Just as unstructured jumps make programs impossible to reason about, unstructured goroutines make concurrent Go programs impossible to reason about.

Golang provides its users with the `errgroup` package which is Go's closest equivalent to structured concurrency and it is genuinely good but there exists 4 concrete limitations:

1. **First error only:** If three services fail at once, you see one error. The other two disappear
   silently
2. **No typed results:** Every task must smuggle results out through closure captures. With generics available since Go 1.18, this is an unnecessary constraint.
3. **No supervision :** For long-running workers, every Go service reinvents a restart loop from
   scratch. There is no standard pattern.
4. **No panic recovery:** A panic in any goroutine kills the entire process. errgroup does not recover
   panics. (sourcegraph/conc fills this gap but inherits the first three.)
