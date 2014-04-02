package main

import (
	"fmt"
)

func calculate() int {
	b := 42
	if b > 0 || b == -42 {
		return -1
	}
	return 0
}

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
	fmt.Println(float64(calculate()))
}
