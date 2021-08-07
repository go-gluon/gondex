package gondex

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// var standardPackages = make(map[string]struct{})

// func init() {
// 	pkgs, err := packages.Load(nil, "std", "golang.org/x/...")
// 	if err != nil {
// 		panic(err)
// 	}
// 	for _, pkg := range pkgs {
// 		standardPackages[pkg.PkgPath] = struct{}{}
// 	}
// }

// func IsStandardPackage(pkg string) bool {
// 	_, ok := standardPackages[pkg]
// 	return ok
// }

type AnnotationInfo struct {
	Name   string
	Params map[string]string
}

type StructInfo struct {
	pkg         *PackageInfo
	named       *types.Named
	data        *types.Struct
	annotations []*AnnotationInfo
}

func (s *StructInfo) Annotations() []*AnnotationInfo {
	return s.annotations
}

func (s *StructInfo) Name() string {
	return s.named.Obj().Name()
}

func (s *StructInfo) Id() string {
	return id(s.pkg, s.named)
}

type FunctionInfo struct {
	pkg       *PackageInfo
	signature *types.Signature
	data      *types.Func
}

type InterfaceInfo struct {
	pkg   *PackageInfo
	named *types.Named
	data  *types.Interface
}

func (s *InterfaceInfo) Id() string {
	return id(s.pkg, s.named)
}

func (s *InterfaceInfo) Name() string {
	return s.named.Obj().Name()
}

type PackageInfo struct {
	data       *packages.Package
	structs    []*StructInfo
	functions  []*FunctionInfo
	interfaces []*InterfaceInfo
}

func (p *PackageInfo) ID() string {
	return p.data.ID
}

type Indexer struct {
	packages []*PackageInfo
	cacheP   map[string]*PackageInfo
	cacheI   map[string]*InterfaceInfo
	cacheS   map[string]*StructInfo
	cacheA   map[string][]*StructInfo
}

func (indexer *Indexer) createPackage(pkg *packages.Package) *PackageInfo {
	p := &PackageInfo{
		data:       pkg,
		structs:    []*StructInfo{},
		functions:  []*FunctionInfo{},
		interfaces: []*InterfaceInfo{},
	}

	indexer.cacheP[p.ID()] = p
	indexer.packages = append(indexer.packages, p)
	return p
}

func (indexer *Indexer) createStructInfo(pkg *PackageInfo, named *types.Named, data *types.Struct, comment *ast.CommentGroup) *StructInfo {
	s := &StructInfo{
		pkg:         pkg,
		named:       named,
		data:        data,
		annotations: []*AnnotationInfo{},
	}
	pkg.structs = append(pkg.structs, s)
	indexer.cacheS[s.Id()] = s

	anno := createAnnotations(comment)
	if anno != nil {
		s.annotations = append(s.annotations, anno...)
		for _, a := range anno {
			tmp := indexer.cacheA[a.Name]
			if tmp == nil {
				tmp = []*StructInfo{}
			}
			tmp = append(tmp, s)
			indexer.cacheA[a.Name] = tmp
		}
	}

	return s
}

func (indexer *Indexer) createInterfaceInfo(pkg *PackageInfo, named *types.Named, data *types.Interface) *InterfaceInfo {
	s := &InterfaceInfo{
		pkg:   pkg,
		named: named,
		data:  data,
	}
	pkg.interfaces = append(pkg.interfaces, s)
	indexer.cacheI[s.Id()] = s
	return s
}

func (indexer *Indexer) createFunctionInfo(pkg *PackageInfo, signature *types.Signature, data *types.Func) *FunctionInfo {
	f := &FunctionInfo{
		pkg:       pkg,
		signature: signature,
		data:      data,
	}
	pkg.functions = append(pkg.functions, f)
	return f
}

func (indexer *Indexer) loadPackages(pattern string) ([]*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedName | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("loading packages for inspection: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("loading packages with errors")
	}
	return pkgs, nil
}

func (indexer *Indexer) Load() error {
	return indexer.LoadPattern("./...")
}

func (indexer *Indexer) LoadPattern(pattern string) error {
	// load packages
	pkgs, err := indexer.loadPackages(pattern)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {

		pkgInfo := indexer.createPackage(pkg)

		comments := findTypeComments(pkg)

		for _, name := range pkg.Types.Scope().Names() {
			obj := pkg.Types.Scope().Lookup(name)

			comment := comments[obj.Name()]

			switch objT := obj.Type().(type) {
			case *types.Named:
				switch undT := objT.Underlying().(type) {
				case *types.Struct:
					indexer.createStructInfo(pkgInfo, objT, undT, comment)
				case *types.Interface:
					indexer.createInterfaceInfo(pkgInfo, objT, undT)
				default:
					panic(fmt.Errorf("not supported named type %v - %T", undT, undT))
				}
			case *types.Signature:
				indexer.createFunctionInfo(pkgInfo, objT, obj.(*types.Func))
			default:
				panic(fmt.Errorf("not supported object type %v - %T", objT, objT))
			}
		}
	}

	return nil
}

func (indexer *Indexer) FindStructByAnnotation(name string) []*StructInfo {
	return indexer.cacheA[name]
}

func (indexer *Indexer) FindInterfaceImplementation(name string) []*StructInfo {
	interfaceInfo := indexer.cacheI[name]
	if interfaceInfo == nil {
		return nil
	}

	result := []*StructInfo{}
	for _, s := range indexer.cacheS {
		if types.AssertableTo(interfaceInfo.data, s.named.Obj().Type()) {
			result = append(result, s)
		}
	}
	return result
}

func CreateIndexer() *Indexer {
	return &Indexer{
		packages: []*PackageInfo{},
		cacheP:   map[string]*PackageInfo{},
		cacheI:   map[string]*InterfaceInfo{},
		cacheS:   map[string]*StructInfo{},
		cacheA:   map[string][]*StructInfo{},
	}

}

func id(pkg *PackageInfo, named *types.Named) string {
	return pkg.ID() + "." + named.Obj().Name()
}

func createAnnotations(comment *ast.CommentGroup) []*AnnotationInfo {
	if comment == nil {
		return nil
	}

	if len(comment.List) == 0 {
		return nil
	}

	result := []*AnnotationInfo{}

	for _, c := range comment.List {
		if strings.HasPrefix(c.Text, "// @") {
			tmp := strings.Split(c.Text[4:], " ")
			if len(tmp) > 0 {
				anno := &AnnotationInfo{
					Name:   tmp[0],
					Params: map[string]string{},
				}
				for i := 1; i < len(tmp); i++ {
					param := strings.Split(tmp[i], "=")
					anno.Params[param[0]] = param[1]
				}
				result = append(result, anno)
			}
		}
	}
	return result
}

func findTypeComments(pkg *packages.Package) map[string]*ast.CommentGroup {
	result := map[string]*ast.CommentGroup{}
	for _, syntax := range pkg.Syntax {
		for _, decl := range syntax.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok {
				if gd.Doc != nil && len(gd.Doc.List) > 0 {
					if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
						result[ts.Name.Name] = gd.Doc
					}
				}
			}
		}
	}
	return result
}
