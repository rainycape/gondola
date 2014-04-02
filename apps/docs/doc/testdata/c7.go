package main

import (
	"fmt"
)

type Something struct {
}

func (s *Something) calculate() int {
	b := 42
	if b > 0 || b == -42 {
		return -1
	}
	return 0
}

func getSomething() *Something {
	return &Something{}
}

func main() {
	a := 7
	switch {
	case a > 0 && a < 10:
		fmt.Println("0-10")
	case a > 50 && a < 100:
		fmt.Println("50-100")
	}
	s1 := new(Something)
	s1.calculate()
	s2 := getSomething()
	fmt.Println(float64(s2.calculate()))
}
