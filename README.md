### ioretry

`go get github.com/mc12d/ioretry@v0.1.0`

Trying-to-be-convenient API for performing retries and timeouts on multiple
IO-like routines that support golang context propagation

Tiny library, no dependencies

### Usage

```go
newF := ioretry.WrapFunc(f, opts...)
```

#### Retry

Function

```go
import "github.com/mc12d/ioretry"

// implement ioretry.IOFunc
f := func(context.Context) error { //... }

newf := ioretry.WrapFunc(f, ioretry.OptRetry(10, 500 * time.Millisecond))

// ...
f(ctx)
```

Retry until `f` succeeds

```go
newF := ioretry.WrapFunc(f, ioretry.OptRetry(ioretry.Forever, time.Second))
```

Repeat forever (will continue repeating if `f` succeeds)

```go
newF := ioretry.WrapFunc(f, ioretry.OptRepeat(ioretry.Forever, time.Second))
```

No time limit

```go
newF := ioretry.WrapFunc(f, ioretry.OptRetry(10, ioretry.OutATime))
```

Using ioretry.IO interface

```go
io := IOImpl{/*...*/}

newIO := ioretry.Wrap(io, OptTimeout(time.Second))
```

#### Multi

Run multiple `IOFunc`s and wait for all to complete

```go
newF := ioretry.MultiFunc(f1, f2, f3)
```

Cancel others and return if any `f_i` returns error

```go
newF := ioretry.MultiFuncFailFast(f1, f2, f3)
```

#### Compose it

```go
newF := ioretry.MultiFunc(
    ioretry.WrapFunc(f1, ioretry.OptRetry(5, 2*time.Second)),
    ioretry.WrapFunc(f2, ioretry.OptTimeout(10, time.Second))
    ioretry.WrapFunc(f3)
)

newIO := ioretry.Multi(
    ioretry.Wrap(io1,
        ioretry.OptRetry(5, 2*time.Second),
        ioretry.OptRecoverPanic(true),
    ),
    ioretry.Wrap(io2, ioretry.OptTimeout(10, time.Second))
    io3
)
```

#### Context

You can cancel context you pass in wrapped func or IO, it works as expected (all
subroutines will be canceled with no more retries/repeats)

#### Errors

Wrapped instances will wrap underlying errors (if any) to `ioretry.MultiError`
or `ioretry.MultiIOError`

Example

```go
newF := ioretry.MultiFunc(
    ioretry.WrapFunc(f1, ioretry.OptRetry(5, 2*time.Second)),
    ioretry.WrapFunc(f2, ioretry.OptTimeout(10, time.Second))
    ioretry.WrapFunc(f3)
)

if err := newF(ctx); err != nil {
    err, _ := err.(ioretry.MultiError)
    for e := range(err) {
        log.Error(e)
    }
}
```

When using `ioretry.IO` interfaces, `ioretry.Multi` returns more convinient
`ioretry.MultiIOError`:

```go
newIO := ioretry.Multi(
    ioretry.Wrap(io1,
        ioretry.OptRetry(5, 2*time.Second),
        ioretry.OptRecoverPanic(true),
    ),
    ioretry.Wrap(io2, ioretry.OptTimeout(10, time.Second))
    io3
)

err := newIO.Run(ctx)

if err := newIO.Run(ctx); err != nil {
    err, _ := err.(ioretry.MultiIOError)
    
    for io, e := range(err) {
        log.Error(fmt.Sprintf("%T: %s", io, e))
    }
}
```
