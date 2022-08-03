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
8. **Until** - execute value until context not done
```go
obs.Pipe(
    rx_go.Until[int](ctx),
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