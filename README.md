`rego` means recycle goroutines and reduce goroutines

## Introduction

Library `rego` is a lightweight goroutine pool implementation that controls concurrent resources by reusing goroutines. Developers can **significantly reduce memory growth** in concurrent programs with minimal performance trade-offs.

## Install

```shell
go get -u github.com/alilestera/rego
```

## Quick Start

### Instantiate with customize capacity

You need to call `rego.New` to instantiate a `Rego` with a given capacity:

```go
r := rego.New(100)
```

### Submit tasks

You can call `r.Submit()` submit tasks:

```go
r.Submit(func() {})
```

### Release

If you need to close `Rego`:

```go
r.Release() // close Rego and ignore all waiting tasks
```

or

```go
r.ReleaseWait() // close Rego and wait all tasks done
```

### Example

```go
package main

import (
	"fmt"

	"github.com/alilestera/rego"
)

func main() {
	r := rego.New(100)
	r.Submit(func() {
		fmt.Println("hello world")
	})
	r.ReleaseWait()
}
```

## Configuration options

| Option          | Default         | Description                              |
| :-------------- | :-------------- | :--------------------------------------- |
| WithMinWorkers  | 0               | Set the minimum number of active workers |
| WithIdleTimeout | 2 * time.Second | Set the idle timeout of active worker    |

### Example

```go
r := rego.New(100, rego.WithMinWorkers(10), rego.WithIdleTimeout(5*time.Second))
```

## Acknowledgments

This project was inspired by or benefited from the following repositories:

- **[gammazero/workerpool](https://github.com/gammazero/workerpool)**
- **[B1NARY-GR0UP/violin](https://github.com/B1NARY-GR0UP/violin)**
- **[panjf2000/ants](https://github.com/panjf2000/ants)**

## License

The source code in `rego` is licensed under [Apache License 2.0](./LICENSE).