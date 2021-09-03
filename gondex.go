package gondex

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	defaultAnnotationRegex = regexp.MustCompile(`^//([0-9A-Za-z_\.]+):([0-9A-Za-z_\.]+)`)
	goPackages             = map[string]struct{}{}
)

func init() {
	pkgs, err := packages.Load(nil, "std", "golang.org/x/...")
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		goPackages[pkg.PkgPath] = struct{}{}
	}
}

func IsGoPackage(pkgPath string) bool {
	_, ok := goPackages[pkgPath]
	return ok
}

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

func (s *StructInfo) Fields(walk FieldStructWalk) {
	f := &FieldStructInfo{
		Info:     s,
		Metadata: map[string]string{},
		Parent:   nil,
		Named:    s.named,
		Struct:   s.data,
		Level:    0,
	}
	walkStruct(f, walk)
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

type IndexerConfig struct {
	DefaultAnnoRegex *regexp.Regexp
	DefaultPattern   []string
	Debug            bool
	SkipGoPackages   bool
}

// Indexer hold the information about the packages and types
type Indexer struct {
	config   *IndexerConfig
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

	indexer.cacheP[p.data.PkgPath] = p
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

	anno := s.ast.Annotations(indexer.config.DefaultAnnoRegex)
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

	anno := s.ast.Annotations(indexer.config.DefaultAnnoRegex)
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

	anno := f.decl.Annotations(indexer.config.DefaultAnnoRegex)
	if anno != nil {
		f.annotations = append(f.annotations, anno...)
	}
	return f
}

// loadPackages load packages
func (indexer *Indexer) loadPackages(pattern ...string) ([]*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.NeedSyntax |
		packages.NeedName |
		packages.NeedTypes |
		// packages.NeedTypesSizes |
		packages.NeedTypesInfo |
		// packages.NeedCompiledGoFiles |
		// packages.NeedDeps |
		packages.NeedImports |
		// packages.NeedExportsFile |
		packages.NeedFiles}
	pkgs, err := packages.Load(cfg, pattern...)
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
	return indexer.LoadPattern(indexer.config.DefaultPattern...)
}

// LoadPattern load packages by the pattern to the indexer
func (indexer *Indexer) LoadPattern(pattern ...string) error {
	// load packages
	pkgs, err := indexer.loadPackages(pattern...)
	if err != nil {
		return err
	}

	// loop over all packages
	for _, pkg := range pkgs {
		indexer.processPackage(pkg)
	}

	return nil
}

func (indexer *Indexer) processPackage(pkg *packages.Package) {
	// check golang package
	if indexer.config.SkipGoPackages && IsGoPackage(pkg.PkgPath) {
		indexer.debug("Skip go pkg: %v", pkg.PkgPath)
		return
	}
	// check if package already process
	if _, e := indexer.cacheP[pkg.PkgPath]; e {
		indexer.debug("Skip read pkg: %v", pkg.PkgPath)
		return
	}

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
				indexer.debug("load pattern not supported named type %v - %T", undT, undT)
			}
		case *types.Signature:
			indexer.createFunctionInfo(pkgInfo, objT, obj.(*types.Func))
		default:
			indexer.debug("load pattern not supported object type %v - %T", objT, objT)
		}
	}

	// check all imports
	if len(pkg.Imports) > 0 {
		for _, v := range pkg.Imports {
			indexer.processPackage(v)
		}
	}
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
		indexer.debug("Interface not found %v", name)
		return nil
	}

	result := []*StructInfo{}
	for _, s := range indexer.cacheS {
		if types.Implements(types.NewPointer(s.named.Obj().Type()), interfaceInfo.data) {
			result = append(result, s)
		} else if types.Implements(s.named.Obj().Type(), interfaceInfo.data) {
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
func CreateDefaultConfig() *IndexerConfig {
	return &IndexerConfig{
		SkipGoPackages:   true,
		Debug:            false,
		DefaultPattern:   []string{"./..."},
		DefaultAnnoRegex: defaultAnnotationRegex,
	}
}

// CreateIndexer creates indexer
func CreateDefaultIndexer() *Indexer {
	return CreateIndexer(CreateDefaultConfig())
}

// CreateIndexer creates indexer
func CreateIndexer(config *IndexerConfig) *Indexer {
	return &Indexer{
		config:   config,
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
func createAnnotations(comment *ast.CommentGroup, r *regexp.Regexp) []*AnnotationInfo {
	if comment == nil {
		return nil
	}

	if len(comment.List) == 0 {
		return nil
	}

	result := []*AnnotationInfo{}

	for _, c := range comment.List {
		sm := r.FindString(c.Text)
		if len(sm) == 0 {
			continue
		}
		an := sm
		tmp := strings.Split(c.Text[len(an)+1:], " ")
		an = strings.TrimSuffix(strings.TrimPrefix(an, "//"), " ")

		anno := &AnnotationInfo{
			Name:   an,
			Params: map[string]string{},
		}

		if len(tmp) > 0 {
			for _, t := range tmp {
				param := strings.Split(t, "=")
				anno.Params[param[0]] = param[1]
			}
		}
		result = append(result, anno)
	}
	return result
}

// AstFuncDecl ast type declaration
type AstTypeDecl struct {
	decl *ast.GenDecl
	ast  *ast.TypeSpec
}

// Annotations returns list of annotations
func (a *AstTypeDecl) Annotations(r *regexp.Regexp) []*AnnotationInfo {
	return createAnnotations(a.decl.Doc, r)
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
func (a *AstFuncDecl) Annotations(r *regexp.Regexp) []*AnnotationInfo {
	return createAnnotations(a.decl.Doc, r)
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

type FieldStructInfo struct {
	Info     *StructInfo
	Parent   *FieldInfo
	Named    *types.Named
	Struct   *types.Struct
	Level    int
	Metadata map[string]string
}

func (f *FieldStructInfo) Name() string {
	if f.Named != nil {
		return f.Named.Obj().Name()
	}
	return f.Parent.Var().Name()
}

func (f *FieldStructInfo) NumFields() int {
	return f.Struct.NumFields()
}

func (f *FieldStructInfo) Var(index int) *types.Var {
	return f.Struct.Field(index)
}

func (f *FieldStructInfo) Tag(index int) string {
	return f.Struct.Tag(index)
}

func (f *FieldStructInfo) Field(index int) *FieldInfo {
	return &FieldInfo{
		Index:    index,
		Struct:   f,
		Metadata: map[string]string{},
	}
}

type FieldInfo struct {
	Struct   *FieldStructInfo
	Index    int
	Metadata map[string]string
}

func (f *FieldInfo) Var() *types.Var {
	return f.Struct.Var(f.Index)
}

func (f *FieldInfo) Tag() string {
	return f.Struct.Tag(f.Index)
}

func (f *FieldInfo) Type() types.Type {
	return f.Var().Type()
}

func (f *FieldInfo) Name() string {
	return f.Var().Name()
}

const tag_empty = ""

func (f *FieldInfo) TagValue(name string) (string, bool) {
	tag := f.Tag()
	if len(tag) == 0 {
		return tag_empty, false
	}
	st := reflect.StructTag(tag)
	return st.Lookup(name)
}

func (f *FieldInfo) FieldStructInfo(named *types.Named, struc *types.Struct) *FieldStructInfo {
	return &FieldStructInfo{
		Parent:   f,
		Metadata: map[string]string{},
		Named:    named,
		Struct:   struc,
		Level:    f.Struct.Level + 1,
	}
}

func walkStruct(struc *FieldStructInfo, walk FieldStructWalk) {

	if !walk.StructBefore(struc) {
		return
	}

	for i := 0; i < struc.NumFields(); i++ {
		field := struc.Field(i)

		walk.FieldBefore(field)

		switch n := field.Type().(type) {
		case *types.Basic:
			walk.Basic(field, n)
		case *types.Slice:
			if walk.Slice(field, n) {
				switch nn := n.Elem().(type) {
				case *types.Struct:
					walkStruct(field.FieldStructInfo(nil, nn), walk)
				case *types.Named:
					switch nnn := nn.Underlying().(type) {
					case *types.Struct:
						walkStruct(field.FieldStructInfo(nn, nnn), walk)
					}
				}
			}
		case *types.Array:
			if walk.Array(field, n) {
				switch nn := n.Elem().(type) {
				case *types.Struct:
					walkStruct(field.FieldStructInfo(nil, nn), walk)
				case *types.Named:
					switch nnn := nn.Underlying().(type) {
					case *types.Struct:
						walkStruct(field.FieldStructInfo(nn, nnn), walk)
					}
				}
			}
		case *types.Map:
			k, v := walk.Map(field, n)
			if k {
				switch nn := n.Key().(type) {
				case *types.Struct:
					walkStruct(field.FieldStructInfo(nil, nn), walk)
				case *types.Named:
					switch nnn := nn.Underlying().(type) {
					case *types.Struct:
						walkStruct(field.FieldStructInfo(nn, nnn), walk)
					}
				}
			}
			if v {
				switch nn := n.Elem().(type) {
				case *types.Struct:
					walkStruct(field.FieldStructInfo(nil, nn), walk)
				case *types.Named:
					switch nnn := nn.Underlying().(type) {
					case *types.Struct:
						walkStruct(field.FieldStructInfo(nn, nnn), walk)
					}
				}
			}
		case *types.Interface:
			walk.Interface(field, nil, n)
		case *types.Struct:
			if walk.Struct(field, nil, n) {
				walkStruct(field.FieldStructInfo(nil, n), walk)
			}
		case *types.Named:
			switch nn := n.Underlying().(type) {
			case *types.Struct:
				if walk.Struct(field, n, nn) {
					walkStruct(field.FieldStructInfo(n, nn), walk)
				}
			case *types.Interface:
				walk.Interface(field, n, nn)
			}
		}

		walk.FieldAfter(field)
	}

	walk.StructAfter(struc)
}

type FieldStructWalk interface {
	FieldBefore(f *FieldInfo) bool
	FieldAfter(f *FieldInfo)
	Basic(f *FieldInfo, t *types.Basic)
	Interface(f *FieldInfo, n *types.Named, t *types.Interface)
	Array(f *FieldInfo, t *types.Array) bool
	Slice(f *FieldInfo, t *types.Slice) bool
	Map(f *FieldInfo, t *types.Map) (bool, bool)
	Struct(f *FieldInfo, n *types.Named, t *types.Struct) bool
	StructBefore(s *FieldStructInfo) bool
	StructAfter(s *FieldStructInfo)
}

func (indexer *Indexer) debug(msg string, a ...interface{}) {
	if !indexer.config.Debug {
		return
	}
	fmt.Printf("[debug] "+msg+"\n", a...)
}
