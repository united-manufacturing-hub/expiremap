# ExpireMap

This go package provides a map with expiring key-value pairs.

## Installation

```bash
go get github.com/united-manufacturing-hub/expiremap
```

## Usage

```go
package main

import (
	"fmt"
	"github.com/united-manufacturing-hub/expiremap/pkg/expiremap"
)

func main() {
	var exMap = expiremap.New[string, string]()
	exMap.Set("key", "value", 10) // 10 seconds
	val := exMap.Get("key")       // "value"
	fmt.Println(val)
	time.Seconds(11)
	val = exMap.Get("key") // nil
    fmt.Println(val)
}
```
