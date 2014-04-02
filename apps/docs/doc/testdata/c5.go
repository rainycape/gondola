package main

import (
	"fmt"
)

func main() {
	defer func() {
		if recover() != nil {
			fmt.Println("recovered")
		}
	}()
	a := 7
	switch {
	case a > 0 && a < 10:
		fmt.Println("0-10")
	case a > 50 && a < 100:
		fmt.Println("50-100")
	}
}
