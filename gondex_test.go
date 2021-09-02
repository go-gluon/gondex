package gondex

import (
	"fmt"
	"go/types"
	"strings"
	"testing"
)

func TestTypes(t *testing.T) {
	indexer := CreateDefaultIndexer()
	indexer.config.Debug = true
	if e := indexer.LoadPattern("github.com/go-gluon/gondex/internal/test/types"); e != nil {
		panic(e)
	}
	s := indexer.Structs()
	if len(s) != 1 {
		panic(fmt.Errorf("No structs found"))
	}
	i := indexer.Interfaces()
	if len(i) != 1 {
		panic(fmt.Errorf("No interfaces found"))
	}
}

func TestAnnotation(t *testing.T) {
	indexer := CreateDefaultIndexer()
	if e := indexer.LoadPattern("github.com/go-gluon/gondex/internal/test", "github.com/go-gluon/gondex/internal/test/project"); e != nil {
		panic(e)
	}
	items := indexer.FindInterfaceByAnnotation("test:test")
	if len(items) == 0 {
		panic(fmt.Errorf("No items found"))
	}
}

func TestFieldStructWalk(t *testing.T) {
	indexer := CreateDefaultIndexer()
	if e := indexer.LoadPattern("github.com/go-gluon/gondex/internal/test", "github.com/go-gluon/gondex/internal/test/project"); e != nil {
		panic(e)
	}

	w := &ExampleFieldWalk{}

	tmp := indexer.FindStructByAnnotation("test:test")
	if len(tmp) == 0 {
		panic(fmt.Errorf("No items found"))
	}
	for _, t := range tmp {
		t.Fields(w)
	}
}

type ExampleFieldWalk struct {
	space string
}

func (e *ExampleFieldWalk) FieldBefore(f *FieldInfo) bool {
	// ignore
	return true
}

func (e *ExampleFieldWalk) FieldAfter(f *FieldInfo) {
	// ignore
}

func (e *ExampleFieldWalk) Basic(f *FieldInfo, t *types.Basic) {
	fmt.Printf("%v%v %v.%v %v\n", f.Struct.Level, e.space, f.Struct.Name(), f.Name(), t)
}

func (e *ExampleFieldWalk) Interface(f *FieldInfo, n *types.Named, t *types.Interface) {
	fmt.Printf("%v%v %v.%v %v\n", f.Struct.Level, e.space, f.Struct.Name(), f.Name(), t)
}

func (e *ExampleFieldWalk) Array(f *FieldInfo, t *types.Array) bool {
	fmt.Printf("%v%v %v.%v %v\n", f.Struct.Level, e.space, f.Struct.Name(), f.Name(), t)
	return true
}

func (e *ExampleFieldWalk) Slice(f *FieldInfo, t *types.Slice) bool {
	fmt.Printf("%v%v %v.%v %v\n", f.Struct.Level, e.space, f.Struct.Name(), f.Name(), t)
	return true
}

func (e *ExampleFieldWalk) Map(f *FieldInfo, t *types.Map) (bool, bool) {
	fmt.Printf("%v%v %v.%v %v\n", f.Struct.Level, e.space, f.Struct.Name(), f.Name(), t)
	return true, true
}

func (e *ExampleFieldWalk) Struct(f *FieldInfo, n *types.Named, t *types.Struct) bool {
	fmt.Printf("%v%v %v.%v %T\n", f.Struct.Level, e.space, f.Struct.Name(), f.Name(), t)
	return true
}

func (e *ExampleFieldWalk) StructBefore(s *FieldStructInfo) bool {
	e.space = e.space + "    "
	return true
}

func (e *ExampleFieldWalk) StructAfter(s *FieldStructInfo) {
	e.space = strings.TrimSuffix(e.space, "    ")
}
