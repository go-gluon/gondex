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

type Struct2 struct {
	name string
}

func (s *Struct2) Name2() string {
	return s.name
}

func TestMethod() {
	test := []Struct{}
	for _, s := range test {
		s.name = "1"
	}
	fmt.Printf("%v\n", test)
}
