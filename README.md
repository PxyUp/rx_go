# RxGo with generics (v1.18+)

```bash
go get github.com/PxyUp/rx_go
```

This attempt to build generic version of Rx(Reactive) library

# Observables
1. **New** - create new observable from observer
```go
observer := rx_go.NewObserver[Y]()
observable := rx_go.New(observer)
```
2. **From** - create new observable from static array
```go
observable := rx_go.From([]int{1, 2, 3})
```
3. **NewInterval** -  return observable from IntervalObserver observer
```go
// Create interval which start from now
interval := rx_go.NewInterval(time.Second, true)
```
4. **NewHttp** - return Observable from HttpObserver
```go
obs, err := rx_go.NewHttp(http.DefaultClient, req)
```
5. **MapTo** - create new observable with modified values
```go
rx_go.MapTo[int, string](rx_go.From([]int{1, 2, 3}), func(t int) string {
	return fmt.Sprintf("hello %d", t)
}).Subscribe()
```
6. **Merge** - merging multi observables with same type into single one
```go
rx_go.Merge[int](rx_go.From[int]([]int{1, 2, 3, 7}...), rx_go.From[int]([]int{4, 5, 6}...)).Subscribe()
```
7. **FromChannel** - create new observable from readable channel
```go
rx_go.FromChannel[int](intChannel).Subscribe()
```
8. **Switch** - change stream for observable
```go
rx_go.Switch(rx_go.From([]int{1, 2, 3}...), func(value int) *rx_go.Observable[string] {
	return rx_go.From(fmt.Sprintf("HELLO %d", value)).Pipe(rx_go.Repeat[string](2))
}).Subscribe()
```
9. **Of** - create static observable with one value
```go
rx_go.Of("hello").Subscribe()
```
10. **Concat** - create static observable witch emit single array of all values
```go
rx_go.Concat(rx_go.From([]int{1, 2, 3, 4, 5, 6}...)).Subscribe()
```
11. **Reduce** - create new observable which return accumulation value from all previous emitted items
```go
rx_go.Reduce(rx_go.From([]int{1, 2, 3, 4, 5, 6}...), func(y string, t int) string {
	return y + fmt.Sprintf("%d", t)
}, "").Subscribe()
```
12. **Pairwise** - create new observable with groups pairs of consecutive emissions together and emits them as an array of two values.
```go
rx_go.Pairwise[int](rx_go.From([]int{1, 2, 3, 4, 5, 6}...)).Subscribe()
```
13. **Never** - observable which never emiting
```go
rx_go.Never
```

# Methods
1. **Subscribe** - create subscription channel and cancel function
```go
ch, cancel := obs.Subscribe()
```
2. **Pipe** - function for accept operators

# Operators:
1. **Filter** - filter out
```go
obs.Pipe(rx_go.Filter[int](func(value int) bool {
	return value > 16
})).Subscribe()
```
2. **Map** - change value
```go
obs.Pipe(rx_go.Map[int](func(value int) int {
	return value * 3
})).Subscribe()
```
3. **LastOne** - get last one from the stream
```go
obs.Pipe(rx_go.LastOne[int]()).Subscribe()
```
4. **FirstOne** - get first one from the stream
```go
obs.Pipe(rx_go.FirstOne[int]()).Subscribe()
```
5. **Delay** - delay before emit next value
```go
obs.Pipe(rx_go.Delay[int](time.Second)).Subscribe()
```
6. **Debounce** - emit value if in provided amount of time new value was not emitted
```go
obs.Pipe(rx_go.Debounce[int](time.Millisecond*500)).Subscribe()
```
7. **Do** - execute action on each value
```go
obs.Pipe(
    rx_go.Do(func(value int) {
        if value == 2 {
            cancel()
        }
    }),
).Subscribe()
```
8. **UntilCtx** - emit value until context not done
```go
obs.Pipe(
    rx_go.UntilCtx[int](ctx),
).Subscribe()
```
9. **Distinct** - execute value if they different from previous
```go
obs.Pipe(rx_go.Distinct[int]()).Subscribe()
```
10. **DistinctWith** - same like Distinct but accept function like comparator
```go
obs.Pipe(rx_go.DistinctWith[int](func(a, b int) bool { return a == b })).Subscribe()
```
11. **Take** - take provided amount from observable
```go
obs.Pipe(rx_go.Take[int](3)).Subscribe()
```
12. **Repeat** - emit value multiple times
```go
rx_go.From(values...).Pipe(rx_go.Repeat[int](2)).Subscribe()
```
13. **AfterCtx** - emit value after ctx is done(values not ignored, they are not emitted)
```go
obs.Pipe(
    rx_go.AfterCtx[int](ctx),
).Subscribe()
```
14. **Skip** - that skips the first count items emitted
```go
rx_go.From([]int{1, 2, 3}...).Pipe(rx_go.Skip[int](2)).Subscribe()
```
15. **StartWith** -  start emitting with predefined value
```go
rx_go.From([]int{1}...).Pipe(rx_go.StartWith(2)).Subscribe()
```
16. **EndWith** -  end emitting with predefined value
```go
rx_go.From([]int{1}...).Pipe(rx_go.EndWith(2)).Subscribe()
```
17. **SkipUntilCtx** - skips items emitted by the Observable until a ctx not done
```go
rx_go.From([]int{1, 2, 3}...).Pipe(
	rx_go.SkipUntilCtx[int](ctx),
).Subscribe()
```
18. **SkipUntil** - skips items emitted by the Observable until a second Observable emits an item(at least one).
```go
rx_go.From([]int{1, 2, 3}...).Pipe(
    rx_go.AfterCtx[int](ctx),
    rx_go.SkipUntil[int, int](rx_go.Of(1).Pipe(rx_go.Do(func(value int) {
        cancelCtx()
    }))),
).Subscribe()
```