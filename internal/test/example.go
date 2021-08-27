package test

import (
	"github.com/go-gluon/gondex/internal/test/project"
	p "github.com/go-gluon/gondex/internal/test/project"
)

type Embedded struct {
	E string `test:"e"`
}

type Special struct {
	Name   string `test:"name"`
	Option string `test:"option"`
}

type Special2 struct {
	Name    string             `test:"name"`
	List    []string           `test:"list-string"`
	Options map[string]Special `test:"options"`
}

type TestI interface {
	test() string
}

// @test:test
type UserTest struct {
	Embedded `test:"e"`
	T        TestI               `test:"i"`
	Data     project.ProjectTest `test:"data"`
	Name     string              `test:"name"`
	Password string              `test:"password"`
	Count    int                 `test:"count"`
	Check    bool                `test:"check"`
	Number   float64             `test:"float"`
	Special  Special             `test:"special" json:"s"`
	Address  struct {
		Street  string             `test:"street"`
		Number  int                `test:"number"`
		Options map[string]Special `test:"options"`
	} `test:"address"`
	Options    map[string]Special        `test:"options"`
	ListInt    []int                     `test:"list-int"`
	ListString []string                  `test:"list-string"`
	ListBool   []bool                    `test:"list-bool"`
	ListFloat  []float64                 `test:"list-float64"`
	MapInt     map[string]int            `test:"map-int"`
	MapString  map[string]string         `test:"map-string"`
	MapFloat   map[string]float64        `test:"map-float"`
	MapBool    map[string]bool           `test:"map-bool"`
	Options2   map[string]p.ProjectTest  `test:"options2"`
	Options3   map[p.ProjectTest]Special `test:"options3"`
	MapStruct  map[string]struct {
		Name   string `test:"name"`
		Number int    `test:"number"`
	} `test:"map-struct"`
}
