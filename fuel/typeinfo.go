package main

import (
	"go/ast"
	"go/token"
	"os"
	"reflect"
)

type fieldInfo struct {
	fieldName      string
	typeName       string
	typeStruct     *ast.StructType
	typeStructFile *ast.File
}

type importVisitor struct {
	pkg  *astPkg
	file *ast.File
}

func (iv importVisitor) Visit(node ast.Node) ast.Visitor {
	switch s := node.(type) {
	case *ast.SelectorExpr:
		for _, imp := range iv.file.Imports {
			impName := importName(imp)
			pkgIdent := s.X.(*ast.Ident)
			if impName == pkgIdent.String() {
				newName := iv.pkg.registerImport(impName, importPath(imp))
				pkgIdent.Name = newName
			}
		}
	}

	return iv
}

type astPkg struct {
	*ast.Package
	fset       *token.FileSet
	genImports map[string]string
}

func (p *astPkg) lookup(sym string) (obj *ast.Object, file *ast.File) {
	for _, f := range p.Files {
		obj := f.Scope.Lookup(sym)
		if obj != nil {
			return obj, f
		}
	}

	return nil, file
}

func (p *astPkg) typeObj(file *ast.File, ftyp ast.Expr) (obj *ast.Object, rfile *ast.File) {

	return nil, file
}

func (p *astPkg) stateStruct(file *ast.File, ftyp ast.Expr) (ss *ast.StructType, rfile *ast.File) {
	switch typ := ftyp.(type) {
	case *ast.StarExpr:
		return p.stateStruct(file, typ.X)

	case *ast.StructType:
		return typ, file

	case *ast.Ident:
		if typ.Obj == nil {
			typ.Obj, file = p.lookup(typ.Name)
		}

		if typ.Obj != nil {
			if spec, ok := typ.Obj.Decl.(*ast.TypeSpec); ok {
				if spec.Type != nil {
					switch et := spec.Type.(type) {
					case *ast.StructType:
						return et, file
					}
				}
			}
		}

	}

	return nil, file
}

func (p *astPkg) registerImport(name string, path string) string {
	newName := name
	var ok bool
	var i int
	for {
		existing := p.genImports[newName]
		ok = existing == "" || existing == path
		if ok {
			break
		}

		i++
		newName = name + string(i)
	}

	p.genImports[newName] = path
	return newName
}

func (p *astPkg) typeName(ftyp ast.Expr, file *ast.File) (string, error) {
	pos := p.fset.Position(ftyp.Pos())
	end := p.fset.Position(ftyp.End())
	of, err := os.Open(pos.Filename)
	if err != nil {
		return "", err
	}

	buf := make([]byte, end.Offset-pos.Offset)
	_, err = of.ReadAt(buf, int64(pos.Offset))
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func anonFieldName(typ ast.Expr) string {
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return anonFieldName(t.X)
	case *ast.SelectorExpr:
		return anonFieldName(t.Sel)
	}

	panic(sfmt("Unhandled ast expression type %T", typ))
	return ""
}

func fieldName(f *ast.Field) string {
	if len(f.Names) > 0 {
		return f.Names[0].Name
	}

	return anonFieldName(f.Type)
}

func (p *astPkg) getStateField(fields []*ast.Field, file *ast.File) (
	state *fieldInfo, err error) {

	var stateField *ast.Field
	for _, f := range fields {
		if f.Tag != nil {
			var stag = reflect.StructTag(f.Tag.Value[1 : len(f.Tag.Value)-1])

			if sf := stag.Get("fuel"); sf == "state" {
				if stateField != nil {
					err = efmt("component can only have 1 state field.")
					return
				}

				stateField = f
			}
		}
	}

	if stateField == nil {
		return nil, nil
	}

	fname := fieldName(stateField)
	ss, tsfile := p.stateStruct(file, stateField.Type)
	visitor := importVisitor{
		pkg:  p,
		file: file,
	}

	ast.Walk(visitor, stateField)
	if ss != nil {
		ast.Walk(visitor, ss)
	}

	typeName, err := p.typeName(stateField.Type, file)
	if err != nil {
		return nil, err
	}

	if typeName[0] != '*' {
		return nil, efmt("illegal type %v for state field %v: state field must be a pointer",
			typeName, fname)
	}

	return &fieldInfo{
		fieldName:      fname,
		typeName:       typeName,
		typeStruct:     ss,
		typeStructFile: tsfile,
	}, nil
}
