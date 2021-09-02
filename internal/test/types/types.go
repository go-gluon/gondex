package types

var test []Struct

type Struct struct {
	Name string
}

type Interface interface {
	Name() string
}
