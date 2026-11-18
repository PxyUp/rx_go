# rx_go

A generic, context-aware ReactiveX-style library for Go 1.18+.

```bash
go get github.com/PxyUp/rx_go
```

Observables are lazy and cold by default: every subscription receives an
independent execution. `Subject`, `BehaviorSubject`, `ReplaySubject`, and
`Share` provide explicit hot/multicast behavior.

## Basic pipeline

```go
values, cancel := rx_go.From(1, 2, 3, 4).
    Pipe(
        rx_go.Filter(func(v int) bool { return v%2 == 0 }),
        rx_go.Map(func(v int) int { return v * 10 }),
    ).
    Subscribe()
defer cancel()

for value := range values {
    fmt.Println(value) // 20, 40
}
```

`Subscribe` is the channel convenience API. Use `SubscribeObserver` to handle
all three Rx notifications:

```go
sub := source.SubscribeObserver(
    ctx,
    func(value int) { fmt.Println(value) },
    func(err error) { log.Println("stream failed:", err) },
    func() { fmt.Println("complete") },
)
defer sub.Cancel()
<-sub.Done()
```

Cancellation belongs to one subscription and propagates upstream through its
context. It does not complete the Observable or affect other subscribers.

## Creation

- `Create`, `Defer`, `From`, `Of`, `FromChannel`
- `EmptyOf`, `NeverOf`, `Throw`
- `NewInterval`, `Timer`, `NewHttp`
- `New(observer)` for a manually driven hot observer

`NewHttp` executes lazily for each subscriber, follows subscription
cancellation, treats non-2xx responses as `HTTPError`, and sends transport and
body errors through the Observable error notification.

## Operators

Same-type operators compose with `Pipe`:

- Transform: `Map`, `Scan`, `Do`
- Filter/select: `Filter`, `Take`, `Skip`, `FirstOne`, `LastOne`, `Find`,
  `ElementAt`, `Distinct`, `DistinctUntilChanged`
- Lifecycle/error: `Finally`, `Catch`, `Retry`, `Timeout`
- Time: `Delay`, `InitialDelay`, `Debounce`, `Throttle`
- Sequence: `StartWith`, `EndWith`, `Repeat`, `UntilCtx`, `AfterCtx`,
  `SkipUntilCtx`, `SkipUntil`

Type-changing and combining functions are standalone because Go methods cannot
introduce a new type parameter:

- `MapTo`, `ScanTo`, `Reduce`, `ToSlice`, `Count`
- `FlatMap`, `ConcatMap`, `Switch`
- `Merge`, `Concat`, `Zip`, `CombineLatest`, `ForkJoin`, `Pairwise`

The names follow standard Rx meanings: `Reduce` emits once, `Scan` emits every
accumulator, `Switch` cancels the previous inner stream, `Concat` concatenates
streams, and `Repeat` repeats the complete source sequence.

## Hot streams

```go
subject := rx_go.NewBehaviorSubject(0)
stream := subject.Observable()

subject.Next(1)
subject.Next(2)
subject.Complete()
```

- `Subject`: live multicast notifications
- `BehaviorSubject`: current value plus live notifications
- `ReplaySubject(capacity)`: bounded replay plus live notifications
- `Share`: ref-counted single upstream subscription

## Development checks

```bash
go test ./...
go test -race ./...
go vet ./...
```

The test suite covers repeat subscriptions, subscription-local cancellation,
terminal errors, operator semantics, timing, HTTP laziness, multicasting, and
concurrent use under the race detector.
