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

// AnnotationInfo represents annotation
type AnnotationInfo struct {
	Name   string
	Params map[string]string
}

// StructInfo information about the struct
type StructInfo struct {
	pkg         *PackageInfo
	named       *types.Named
	data        *types.Struct
	ast         *AstTypeDecl
	annotations []*AnnotationInfo
}

// Ast ast declaration of the type
func (s *StructInfo) Ast() *AstTypeDecl {
	return s.ast
}

// Package struct package info
func (s *StructInfo) Package() *PackageInfo {
	return s.pkg
}

// Named type named of struct
func (s *StructInfo) Named() *types.Named {
	return s.named
}

// Struct type of struct
func (s *StructInfo) Struct() *types.Struct {
	return s.data
}

// Annotations returns list of struct annotations of empty list
func (s *StructInfo) Annotations() []*AnnotationInfo {
	return s.annotations
}

// Name this is the name of the struct
func (s *StructInfo) Name() string {
	return s.named.Obj().Name()
}

// Id of the struct
func (s *StructInfo) Id() string {
	return id(s.pkg, s.named)
}

// FunctionInfo represents function
type FunctionInfo struct {
	pkg         *PackageInfo
	signature   *types.Signature
	data        *types.Func
	decl        *AstFuncDecl
	annotations []*AnnotationInfo
}

// Decl ast declaration of the type
func (s *FunctionInfo) Decl() *AstFuncDecl {
	return s.decl
}

// InterfaceInfo represents interface
type InterfaceInfo struct {
	pkg         *PackageInfo
	named       *types.Named
	data        *types.Interface
	ast         *AstTypeDecl
	annotations []*AnnotationInfo
}

// Ast ast declaration of the type
func (s *InterfaceInfo) Ast() *AstTypeDecl {
	return s.ast
}

// Id of the interface
func (s *InterfaceInfo) Id() string {
	return id(s.pkg, s.named)
}

// Name of the interface
func (s *InterfaceInfo) Name() string {
	return s.named.Obj().Name()
}

// Annotations returns list of interface annotations or emtpy list
func (s *InterfaceInfo) Annotations() []*AnnotationInfo {
	return s.annotations
}

// PackageInfo struct represents the package information
type PackageInfo struct {
	ast        *AstInfo
	data       *packages.Package
	structs    []*StructInfo
	functions  []*FunctionInfo
	interfaces []*InterfaceInfo
}

// Data of the package
func (p *PackageInfo) Data() *packages.Package {
	return p.data
}

// ID of the package
func (p *PackageInfo) Id() string {
	return p.data.ID
}

// Indexer hold the information about the packages and types
type Indexer struct {
	packages []*PackageInfo
	cacheP   map[string]*PackageInfo
	cacheI   map[string]*InterfaceInfo
	cacheS   map[string]*StructInfo
	cacheA   map[string][]*StructInfo
	cacheAI  map[string][]*InterfaceInfo
}

// createPackageInfo creates package info
func (indexer *Indexer) createPackageInfo(pkg *packages.Package) *PackageInfo {
	// load ast info
	ast := processAstInfo(pkg)

	p := &PackageInfo{
		ast:        ast,
		data:       pkg,
		structs:    []*StructInfo{},
		functions:  []*FunctionInfo{},
		interfaces: []*InterfaceInfo{},
	}

	indexer.cacheP[p.Id()] = p
	indexer.packages = append(indexer.packages, p)
	return p
}

// createStructInfo creates struct info
func (indexer *Indexer) createStructInfo(pkg *PackageInfo, named *types.Named, data *types.Struct) *StructInfo {
	name := named.Obj().Name()

	s := &StructInfo{
		pkg:         pkg,
		named:       named,
		data:        data,
		ast:         pkg.ast.types[name],
		annotations: []*AnnotationInfo{},
	}
	pkg.structs = append(pkg.structs, s)
	indexer.cacheS[s.Id()] = s

	if s.ast == nil {
		return s
	}

	anno := s.ast.Annotations()
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

// createInterfaceInfo creates interface info
func (indexer *Indexer) createInterfaceInfo(pkg *PackageInfo, named *types.Named, data *types.Interface) *InterfaceInfo {
	name := named.Obj().Name()

	s := &InterfaceInfo{
		pkg:   pkg,
		named: named,
		data:  data,
		ast:   pkg.ast.types[name],
	}
	pkg.interfaces = append(pkg.interfaces, s)
	indexer.cacheI[s.Id()] = s

	if s.ast == nil {
		return s
	}

	anno := s.ast.Annotations()
	if anno != nil {
		s.annotations = append(s.annotations, anno...)
		for _, a := range anno {
			tmp := indexer.cacheAI[a.Name]
			if tmp == nil {
				tmp = []*InterfaceInfo{}
			}
			tmp = append(tmp, s)
			indexer.cacheAI[a.Name] = tmp
		}
	}
	return s
}

// createFunctionInfo create function info
func (indexer *Indexer) createFunctionInfo(pkg *PackageInfo, signature *types.Signature, data *types.Func) *FunctionInfo {
	f := &FunctionInfo{
		pkg:       pkg,
		signature: signature,
		data:      data,
		decl:      pkg.ast.functions[data.Name()],
	}
	pkg.functions = append(pkg.functions, f)
	if f.decl == nil {
		return f
	}

	anno := f.decl.Annotations()
	if anno != nil {
		f.annotations = append(f.annotations, anno...)
	}
	return f
}

// loadPackages load packages
func (indexer *Indexer) loadPackages(pattern string) ([]*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.NeedSyntax |
		packages.NeedName |
		packages.NeedTypes |
		// packages.NeedTypesSizes |
		packages.NeedTypesInfo |
		// packages.NeedCompiledGoFiles |
		// packages.NeedDeps |
		// packages.NeedImports |
		// packages.NeedExportsFile |
		packages.NeedFiles}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("loading packages for inspection: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("loading packages with errors")
	}
	return pkgs, nil
}

// Load load packages by default pattern ./...
func (indexer *Indexer) Load() error {
	return indexer.LoadPattern("./...")
}

// LoadPattern load packages by the pattern to the indexer
func (indexer *Indexer) LoadPattern(pattern string) error {
	// load packages
	pkgs, err := indexer.loadPackages(pattern)
	if err != nil {
		return err
	}

	// loop over all packages
	for _, pkg := range pkgs {

		// create package info
		pkgInfo := indexer.createPackageInfo(pkg)

		// loop over all types
		for _, name := range pkg.Types.Scope().Names() {
			obj := pkg.Types.Scope().Lookup(name)

			switch objT := obj.Type().(type) {
			case *types.Named:
				switch undT := objT.Underlying().(type) {
				case *types.Struct:
					indexer.createStructInfo(pkgInfo, objT, undT)
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

// FindStructByAnnotation find all structs by annotation
func (indexer *Indexer) FindStructByAnnotation(name string) []*StructInfo {
	return indexer.cacheA[name]
}

// FindInterfaceByAnnotation find all interfaces by annotation
func (indexer *Indexer) FindInterfaceByAnnotation(name string) []*StructInfo {
	return indexer.cacheA[name]
}

// FindInterfaceImplementation find all interface implementations
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

// Packages return map of all packages
func (indexer *Indexer) Packages() map[string]*PackageInfo {
	return indexer.cacheP
}

// Interfaces return map of all interfaces
func (indexer *Indexer) Interfaces() map[string]*InterfaceInfo {
	return indexer.cacheI
}

// Structs return map of all structs
func (indexer *Indexer) Structs() map[string]*StructInfo {
	return indexer.cacheS
}

// CreateIndexer creates indexer
func CreateIndexer() *Indexer {
	return &Indexer{
		packages: []*PackageInfo{},
		cacheP:   map[string]*PackageInfo{},
		cacheI:   map[string]*InterfaceInfo{},
		cacheS:   map[string]*StructInfo{},
		cacheA:   map[string][]*StructInfo{},
		cacheAI:  map[string][]*InterfaceInfo{},
	}

}

// id generate ID for the named type (struct, interface)
func id(pkg *PackageInfo, named *types.Named) string {
	return pkg.data.PkgPath + "." + named.Obj().Name()
}

// createAnnotations this method creates list of annotations info from the comments
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

// AstFuncDecl ast type declaration
type AstTypeDecl struct {
	decl *ast.GenDecl
	ast  *ast.TypeSpec
}

// Annotations returns list of annotations
func (a *AstTypeDecl) Annotations() []*AnnotationInfo {
	return createAnnotations(a.decl.Doc)
}

// GenDecl struct type of the type
func (a *AstTypeDecl) GenDecl() *ast.GenDecl {
	return a.decl
}

// StructType struct type of the type
func (a *AstTypeDecl) StructType() *ast.StructType {
	return a.ast.Type.(*ast.StructType)
}

// InterfaceType struct type of the type
func (a *AstTypeDecl) InterfaceType() *ast.InterfaceType {
	return a.ast.Type.(*ast.InterfaceType)
}

// AstFuncDecl ast function declaration
type AstFuncDecl struct {
	decl *ast.FuncDecl
}

// Annotations returns list of annotations
func (a *AstFuncDecl) Annotations() []*AnnotationInfo {
	return createAnnotations(a.decl.Doc)
}

// FuncType struct type of the type
func (a *AstFuncDecl) FuncType() *ast.FuncType {
	return a.decl.Type
}

// AstInfo syntax info
type AstInfo struct {
	functions map[string]*AstFuncDecl
	types     map[string]*AstTypeDecl
}

// processAstInfo find all types and functions in the AST
func processAstInfo(pkg *packages.Package) *AstInfo {
	result := &AstInfo{
		functions: map[string]*AstFuncDecl{},
		types:     map[string]*AstTypeDecl{},
	}
	for _, syntax := range pkg.Syntax {
		for _, decl := range syntax.Decls {
			switch dt := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range dt.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						result.types[ts.Name.Name] = &AstTypeDecl{decl: dt, ast: ts}
					}
				}
			case *ast.FuncDecl:
				result.functions[dt.Name.Name] = &AstFuncDecl{decl: dt}
			default:
				panic(fmt.Errorf("not supported decl type %v - %T", dt, dt))
			}
		}
	}
	return result
}
