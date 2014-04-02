package main

import (
	"fmt"
)

func main() {
	a := 7
	switch a {
	case 0:
		fmt.Println("0")
	case 1:
		fmt.Println("1")
	case 2:
		fmt.Println("2")
	default:
		fmt.Println("something else")
	}
}
