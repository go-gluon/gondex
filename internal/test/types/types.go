package types

import "fmt"

var list []Struct

func init() {
	fmt.Printf("%v\n", list)
}

type Struct struct {
	Name string
}

type Interface interface {
	Name() string
}

func TestMethod() {
	test := []Struct{}
	for _, s := range test {
		s.Name = "1"
	}
	fmt.Printf("%v\n", test)
}
