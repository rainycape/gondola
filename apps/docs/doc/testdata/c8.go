package main

import (
	"fmt"
)

type anotherthing struct {
}

func (a *anotherthing) calculate() int {
	b := 42
	if b > 0 || b == -42 {
		return -1
	}
	return 0
}

type something struct {
	a *anotherthing
}

func (s *something) calculate() int {
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
	s := new(something)
	s.a = new(anotherthing)
	fmt.Println(float64(s.a.calculate()))
}
