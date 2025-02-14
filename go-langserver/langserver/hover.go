package langserver

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	doc "github.com/slimsag/godocmd"
	"github.com/khulnasoft/go/go-langserver/langserver/internal/godef"
	"github.com/khulnasoft/go/go-langserver/langserver/util"
	"github.com/khulnasoft/go/go-lsp"
	"github.com/khulnasoft/go/jsonrpc2"
	"golang.org/x/tools/go/packages"
)

func (h *LangHandler) handleHover(ctx context.Context, conn jsonrpc2.JSONRPC2, req *jsonrpc2.Request, params lsp.TextDocumentPositionParams) (*lsp.Hover, error) {
	if h.config.UseBinaryPkgCache {
		return h.handleHoverGodef(ctx, conn, req, params)
	}

	if !util.IsURI(params.TextDocument.URI) {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("textDocument/hover not yet supported for out-of-workspace URI (%q)", params.TextDocument.URI),
		}
	}

	fset, node, _, prog, pkg, _, err := h.typecheck(ctx, conn, params.TextDocument.URI, params.Position)
	if err != nil {
		// Invalid nodes means we tried to click on something which is
		// not an ident (eg comment/string/etc). Return no information.
		if _, ok := err.(*invalidNodeError); ok {
			return nil, nil
		}
		// This is a common error we get in production when a user is
		// browsing a go pkg which only contains files we can't
		// analyse (usually due to build tags). To reduce signal of
		// actual bad errors, we return no error in this case.
		if _, ok := err.(*build.NoGoError); ok {
			return nil, nil
		}
		return nil, err
	}

	o := pkg.ObjectOf(node)
	t := pkg.TypeOf(node)
	if o == nil && t == nil {
		comments := packageDoc(pkg.Files, node.Name)

		// Package statement idents don't have an object, so try that separately.
		r := rangeForNode(fset, node)
		if pkgName := packageStatementName(fset, pkg.Files, node); pkgName != "" {
			return &lsp.Hover{
				Contents: maybeAddComments(comments, []lsp.MarkedString{{Language: "go", Value: "package " + pkgName}}),
				Range:    &r,
			}, nil
		}
		return nil, fmt.Errorf("type/object not found at %+v", params.Position)
	}

	// Don't package-qualify the string output.
	qf := func(*types.Package) string { return "" }

	// Handle builtin objects with invalid locations.
	if o != nil && !o.Pos().IsValid() {
		contents := builtinDoc(node.Name)

		return &lsp.Hover{
			Contents: contents,
		}, nil
	}

	var s string
	var extra string
	if f, ok := o.(*types.Var); ok && f.IsField() {
		// TODO(sqs): make this be like (T).F not "struct field F string".
		s = "struct " + o.String()
	} else if o != nil {
		if obj, ok := o.(*types.TypeName); ok {
			typ := obj.Type().Underlying()
			if _, ok := typ.(*types.Struct); ok {
				s = "type " + obj.Name() + " struct"
				extra = prettyPrintTypesString(types.TypeString(typ, qf))
			}
			if _, ok := typ.(*types.Interface); ok {
				s = "type " + obj.Name() + " interface"
				extra = prettyPrintTypesString(types.TypeString(typ, qf))
			}
		}
		if s == "" {
			s = objectString(o, qf)
		}
	} else if t != nil {
		s = types.TypeString(t, qf)
	}

	findComments := func(o types.Object) (string, error) {
		if o == nil {
			return "", nil
		}

		// Package names must be resolved specially, so do this now to avoid
		// additional overhead.
		if v, ok := o.(*types.PkgName); ok {
			pkg := prog.Package(v.Imported().Path())
			if pkg == nil {
				return "", fmt.Errorf("failed to import package %q", v.Imported().Path())
			}
			return packageDoc(pkg.Files, node.Name), nil
		}

		// Resolve the object o into its respective ast.Node
		_, path, _ := prog.PathEnclosingInterval(o.Pos(), o.Pos())
		if path == nil {
			return "", nil
		}

		// Pull the comment out of the comment map for the file. Do
		// not search too far away from the current path.
		var comments string
		for i := 0; i < 3 && i < len(path) && comments == ""; i++ {
			switch v := path[i].(type) {
			case *ast.Field:
				// Concat associated documentation with any inline comments
				comments = joinCommentGroups(v.Doc, v.Comment)
			case *ast.ValueSpec:
				comments = v.Doc.Text()
			case *ast.TypeSpec:
				comments = v.Doc.Text()
			case *ast.GenDecl:
				comments = v.Doc.Text()
			case *ast.FuncDecl:
				comments = v.Doc.Text()
			}
		}
		return comments, nil
	}

	comments, err := findComments(o)
	if err != nil {
		return nil, err
	}
	contents := maybeAddComments(comments, []lsp.MarkedString{{Language: "go", Value: s}})
	if extra != "" {
		// If we have extra info, ensure it comes after the usually
		// more useful documentation
		contents = append(contents, lsp.MarkedString{Language: "go", Value: extra})
	}

	r := rangeForNode(fset, node)
	return &lsp.Hover{
		Contents: contents,
		Range:    &r,
	}, nil
}

// packageStatementName returns the package name ((*ast.Ident).Name)
// of node iff node is the package statement of a file ("package p").
func packageStatementName(fset *token.FileSet, files []*ast.File, node *ast.Ident) string {
	for _, f := range files {
		if f.Name == node {
			return node.Name
		}
	}
	return ""
}

// maybeAddComments appends the specified comments converted to Markdown godoc
// form to the specified contents slice, if the comments string is not empty.
func maybeAddComments(comments string, contents []lsp.MarkedString) []lsp.MarkedString {
	if comments == "" {
		return contents
	}
	var b bytes.Buffer
	doc.ToMarkdown(&b, comments, nil)
	return append(contents, lsp.RawMarkedString(b.String()))
}

// joinCommentGroups joins the resultant non-empty comment text from two
// CommentGroups with a newline.
func joinCommentGroups(a, b *ast.CommentGroup) string {
	aText := a.Text()
	bText := b.Text()
	if aText == "" {
		return bText
	} else if bText == "" {
		return aText
	} else {
		return aText + "\n" + bText
	}
}

// packageDoc finds the documentation for the named package from its files or
// additional files.
func packageDoc(files []*ast.File, pkgName string) string {
	for _, f := range files {
		if f.Name.Name == pkgName {
			txt := f.Doc.Text()
			if strings.TrimSpace(txt) != "" {
				return txt
			}
		}
	}
	return ""
}

// builtinDoc finds the documentation for a builtin node.
func builtinDoc(ident string) []lsp.MarkedString {
	// Grab files from builtin package
	pkgs, err := packages.Load(
		&packages.Config{
			Mode: packages.LoadFiles,
		},
		"builtin",
	)
	if err != nil {
		return nil
	}

	// Parse the files into ASTs
	pkg := pkgs[0]
	fs := token.NewFileSet()
	asts := &ast.Package{
		Name:  "builtin",
		Files: make(map[string]*ast.File),
	}
	for _, filename := range pkg.GoFiles {
		file, err := parser.ParseFile(fs, filename, nil, parser.ParseComments)
		if err != nil {
			fmt.Println(err.Error())
		}
		asts.Files[filename] = file
	}

	// Extract documentation and declaration from the ASTs
	docs := doc.New(asts, "builtin", doc.AllDecls)
	node, pos := findDocIdent(docs, ident)
	contents, _ := fmtDocObject(fs, node, fs.Position(pos))
	return contents
}

// findDocIdentt walks an input *doc.Package and locates the *doc.Value,
// *doc.Type, or *doc.Func with the given identifier.
func findDocIdent(docs *doc.Package, ident string) (node interface{}, pos token.Pos) {
	searchFuncs := func(funcs []*doc.Func) bool {
		for _, f := range funcs {
			if f.Name == ident {
				node = f
				pos = f.Decl.Pos()
				return true
			}
		}
		return false
	}

	searchVars := func(vars []*doc.Value) bool {
		for _, v := range vars {
			for _, spec := range v.Decl.Specs {
				switch t := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range t.Names {
						if name.Name == ident {
							node = v
							pos = name.Pos()
							return true
						}
					}
				}
			}
		}
		return false
	}

	if searchFuncs(docs.Funcs) {
		return
	}

	if searchVars(docs.Consts) {
		return
	}

	if searchVars(docs.Vars) {
		return
	}

	for _, t := range docs.Types {
		if t.Name == ident {
			node = t
			pos = t.Decl.Pos()
			return
		}

		if searchFuncs(t.Funcs) {
			return
		}

		if searchVars(t.Consts) {
			return
		}

		if searchVars(t.Vars) {
			return
		}
	}

	return
}

// commentsToText converts a slice of []*ast.CommentGroup to a flat string,
// ensuring whitespace-only comment groups are dropped.
func commentsToText(cgroups []*ast.CommentGroup) (text string) {
	for _, c := range cgroups {
		if strings.TrimSpace(c.Text()) != "" {
			text += c.Text()
		}
	}
	return text
}

// prettyPrintTypesString is pretty printing specific to the output of
// types.*String. Instead of re-implementing the printer, we can just
// transform its output.
func prettyPrintTypesString(s string) string {
	// Don't bother including the fields if it is empty
	if strings.HasSuffix(s, "{}") {
		return ""
	}
	var b bytes.Buffer
	b.Grow(len(s))
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case ';':
			b.WriteByte('\n')
			for j := 0; j < depth; j++ {
				b.WriteString("    ")
			}
			// Skip following space
			i++

		case '{':
			if i == len(s)-1 {
				// This should never happen, but in case it
				// does give up
				return s
			}

			n := s[i+1]
			if n == '}' {
				// Do not modify {}
				b.WriteString("{}")
				// We have already written }, so skip
				i++
			} else {
				// We expect fields to follow, insert a newline and space
				depth++
				b.WriteString(" {\n")
				for j := 0; j < depth; j++ {
					b.WriteString("    ")
				}
			}

		case '}':
			depth--
			if depth < 0 {
				return s
			}
			b.WriteString("\n}")

		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func (h *LangHandler) handleHoverGodef(ctx context.Context, conn jsonrpc2.JSONRPC2, req *jsonrpc2.Request, params lsp.TextDocumentPositionParams) (*lsp.Hover, error) {
	// First perform the equivalent of a textDocument/definition request in
	// order to resolve the definition position.
	fset, res, _, err := h.definitionGodef(ctx, params)
	if err != nil {
		if err == godef.ErrNoIdentifierFound {
			// This is expected to happen when hovering over
			// comments/strings/whitespace/etc), just return no info.
			return nil, nil
		}
		return nil, err
	}

	// If our target is a package import statement or package selector, then we
	// handle that separately now.
	if res.Package != nil {
		// res.Package.Name is invalid since it was imported with FindOnly, so
		// import normally now.
		bpkg, err := build.Default.ImportDir(res.Package.Dir, 0)
		if err != nil {
			return nil, err
		}

		// Parse the entire dir into its respective AST packages.
		pkgs, err := parser.ParseDir(fset, res.Package.Dir, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		pkg := pkgs[bpkg.Name]

		// Find the package doc comments.
		pkgFiles := make([]*ast.File, 0, len(pkg.Files))
		for _, f := range pkg.Files {
			pkgFiles = append(pkgFiles, f)
		}
		comments := packageDoc(pkgFiles, bpkg.Name)

		return &lsp.Hover{
			Contents: maybeAddComments(comments, []lsp.MarkedString{{Language: "go", Value: fmt.Sprintf("package %s (%q)", bpkg.Name, bpkg.ImportPath)}}),

			// TODO(slimsag): I think we can add Range here, but not exactly
			// sure. res.Start and res.End are only present if it's a package
			// selector, not an import statement. Since Range is optional,
			// we're omitting it here.
		}, nil
	}

	loc := goRangeToLSPLocation(fset, res.Start, res.End)

	// Handle builtin objects with invalid locations.
	if loc.URI == "file://" {
		_, node, _, _, _, _, err := h.typecheck(ctx, conn, params.TextDocument.URI, params.Position)
		if err != nil {
			// Invalid nodes means we tried to click on something which is
			// not an ident (eg comment/string/etc). Return no information.
			if _, ok := err.(*invalidNodeError); ok {
				return nil, nil
			}
			// This is a common error we get in production when a user is
			// browsing a go pkg which only contains files we can't
			// analyse (usually due to build tags). To reduce signal of
			// actual bad errors, we return no error in this case.
			if _, ok := err.(*build.NoGoError); ok {
				return nil, nil
			}
			return nil, err
		}
		contents := builtinDoc(node.Name)
		return &lsp.Hover{
			Contents: contents,
		}, nil
	}

	// convert the path into a real path because 3rd party tools
	// might load additional code based on the file's package
	filename := util.UriToRealPath(loc.URI)

	// Parse the entire dir into its respective AST packages.
	pkgs, err := parser.ParseDir(fset, filepath.Dir(filename), nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Locate the AST package that contains the file we're interested in.
	foundImportPath, foundPackage, err := packageForFile(pkgs, filename)
	if err != nil {
		return nil, err
	}

	// Create documentation for the package.
	docPkg := doc.New(foundPackage, foundImportPath, doc.AllDecls)

	// Locate the target in the docs.
	target := fset.Position(res.Start)
	docObject := findDocTarget(fset, target, docPkg)
	if docObject == nil {
		// probably a local variable, so just ignore.
		return &lsp.Hover{}, nil
	}

	contents, _ := fmtDocObject(fset, docObject, target)
	return &lsp.Hover{
		Contents: contents,
	}, nil
}

// packageForFile returns the import path and pkg from pkgs that contains the
// named file.
func packageForFile(pkgs map[string]*ast.Package, filename string) (string, *ast.Package, error) {
	for path, pkg := range pkgs {
		for pkgFile := range pkg.Files {
			if pkgFile == filename {
				return path, pkg, nil
			}
		}
	}
	return "", nil, fmt.Errorf("failed to find %q in packages %+v", filename, pkgs)
}

// inRange tells if x is in the range of a-b inclusive.
func inRange(x, a, b token.Position) bool {
	if !util.PathEqual(x.Filename, a.Filename) || !util.PathEqual(x.Filename, b.Filename) {
		return false
	}
	return x.Offset >= a.Offset && x.Offset <= b.Offset
}

// findDocTarget walks an input *doc.Package and locates the *doc.Value,
// *doc.Type, or *doc.Func for the given target position.
func findDocTarget(fset *token.FileSet, target token.Position, in interface{}) interface{} {
	switch v := in.(type) {
	case *doc.Package:
		for _, x := range v.Consts {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		for _, x := range v.Types {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		for _, x := range v.Vars {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		for _, x := range v.Funcs {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		return nil
	case *doc.Value:
		if inRange(target, fset.Position(v.Decl.Pos()), fset.Position(v.Decl.End())) {
			return v
		}
		return nil
	case *doc.Type:
		if inRange(target, fset.Position(v.Decl.Pos()), fset.Position(v.Decl.End())) {
			return v
		}
		for _, x := range v.Consts {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		for _, x := range v.Vars {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		for _, x := range v.Funcs {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		for _, x := range v.Methods {
			if r := findDocTarget(fset, target, x); r != nil {
				return r
			}
		}
		return nil
	case *doc.Func:
		if inRange(target, fset.Position(v.Decl.Pos()), fset.Position(v.Decl.End())) {
			return v
		}
		return nil
	default:
		panic("unreachable")
	}
}

// fmtDocObject formats one of:
//
// *doc.Value
// *doc.Type
// *doc.Func
//
func fmtDocObject(fset *token.FileSet, x interface{}, target token.Position) ([]lsp.MarkedString, ast.Node) {
	switch v := x.(type) {
	case *doc.Value: // Vars and Consts
		// Sort the specs by distance to find the one nearest to target.
		sort.Sort(byDistance{v.Decl.Specs, fset, target})
		spec := v.Decl.Specs[0].(*ast.ValueSpec)

		// Use the doc directly above the var inside a var() block, or if there
		// is none, fall back to the doc directly above the var() block.
		doc := spec.Doc.Text()
		if doc == "" {
			doc = v.Doc
		}

		// Create a copy of the spec with no doc for formatting separately.
		cpy := *spec
		cpy.Doc = nil
		value := v.Decl.Tok.String() + " " + fmtNode(fset, &cpy)
		return maybeAddComments(doc, []lsp.MarkedString{{Language: "go", Value: value}}), spec

	case *doc.Type: // Type declarations
		spec := v.Decl.Specs[0].(*ast.TypeSpec)

		// Handle interfaces methods and struct fields separately now.
		switch s := spec.Type.(type) {
		case *ast.InterfaceType:
			// Find the method that is an exact match for our target position.
			for _, field := range s.Methods.List {
				if fset.Position(field.Pos()).Offset == target.Offset {
					// An exact match.
					value := fmt.Sprintf("func (%s).%s%s", spec.Name.Name, field.Names[0].Name, strings.TrimPrefix(fmtNode(fset, field.Type), "func"))
					return maybeAddComments(field.Doc.Text(), []lsp.MarkedString{{Language: "go", Value: value}}), field
				}
			}

		case *ast.StructType:
			// Find the field that is an exact match for our target position.
			for _, field := range s.Fields.List {
				if fset.Position(field.Pos()).Offset == target.Offset {
					// An exact match.
					value := fmt.Sprintf("struct field %s %s", field.Names[0], fmtNode(fset, field.Type))
					// Concat associated documentation with any inline comments
					comments := joinCommentGroups(field.Doc, field.Comment)
					return maybeAddComments(comments, []lsp.MarkedString{{Language: "go", Value: value}}), field
				}
			}
		}

		// Formatting of all type declarations: structs, interfaces, integers, etc.
		name := v.Decl.Tok.String() + " " + spec.Name.Name + " " + typeName(fset, spec.Type)
		res := []lsp.MarkedString{{Language: "go", Value: name}}

		doc := spec.Doc.Text()
		if doc == "" {
			doc = v.Doc
		}
		res = maybeAddComments(doc, res)

		if n := typeName(fset, spec.Type); n == "interface" || n == "struct" {
			res = append(res, lsp.MarkedString{Language: "go", Value: fmtNode(fset, spec.Type)})
		}
		return res, spec

	case *doc.Func: // Functions
		return maybeAddComments(v.Doc, []lsp.MarkedString{{Language: "go", Value: fmtNode(fset, v.Decl)}}), v.Decl
	default:
		panic("unreachable")
	}
}

// typeName returns the name of typ, shortening interface and struct types to
// just "interface" and "struct" rather than their full contents (incl. methods
// and fields).
func typeName(fset *token.FileSet, typ ast.Expr) string {
	switch typ.(type) {
	case *ast.InterfaceType:
		return "interface"
	case *ast.StructType:
		return "struct"
	default:
		return fmtNode(fset, typ)
	}
}

// objectString wraps types.ObjectString for specific type formatting.
func objectString(obj types.Object, qf types.Qualifier) string {
	str := types.ObjectString(obj, qf)
	switch obj := obj.(type) {
	case *types.Const:
		str = fmt.Sprintf("%s = %s", str, obj.Val())
	}
	return str
}

// fmtNode formats the given node as a string.
func fmtNode(fset *token.FileSet, n ast.Node) string {
	var buf bytes.Buffer
	err := format.Node(&buf, fset, n)
	if err != nil {
		panic("unreachable")
	}
	return buf.String()
}

// byDistance sorts specs by distance to the target position.
type byDistance struct {
	specs  []ast.Spec
	fset   *token.FileSet
	target token.Position
}

func (b byDistance) Len() int      { return len(b.specs) }
func (b byDistance) Swap(i, j int) { b.specs[i], b.specs[j] = b.specs[j], b.specs[i] }
func (b byDistance) Less(ii, jj int) bool {
	i := b.fset.Position(b.specs[ii].Pos())
	j := b.fset.Position(b.specs[jj].Pos())
	return abs(b.target.Offset-i.Offset) < abs(b.target.Offset-j.Offset)
}

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
