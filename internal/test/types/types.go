package types

import "fmt"

var list []Struct

func init() {
	fmt.Printf("%v\n", list)
}

type Struct struct {
	name string
}

func (s *Struct) Name() string {
	return s.name
}

type Interface interface {
	Name() string
}

func TestMethod() {
	test := []Struct{}
	for _, s := range test {
		s.name = "1"
	}
	fmt.Printf("%v\n", test)
}
