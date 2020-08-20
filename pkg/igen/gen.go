package igen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// PropogateInterfaces wraps the given interface in the given package to allow also
// implementing a named set of additional interfaces iff the given
// wrappingInterface also implements them, as determined at runtime.
//
// Let's consider a simple concrete example:
// Assume we have a package with the following struct:
//
//     type foo struct {
//         io.Reader
//     }
//
// One can call this package with:
//
//     PropogateInterfaces(pkg, "propogateReader", "f *foo.Reader", []string{"io.Closer"})
//
// Which will generate code for:
//
//     func (f *foo) propogateReader() io.Reader {
//        // returns a type that implements 'io.Closer' iff f.Reader implements 'io.Closer'
//     }
//
// Why is this ever useful? See https://medium.com/@cep21/interface-wrapping-method-erasure-c523b3549912
func PropogateInterfaces(
	pkg *packages.Package,
	wrapperFuncName string,
	structSelector string,
	wrappedInterfaces []string,
) (string, error) {
	structSel, err := parseStructSel(pkg, structSelector)
	if err != nil {
		return "", err
	}

	// And now look up all the interfaces we're supposed to wrap
	wrappingIfaces := make([]*iface, 0, len(wrappedInterfaces))
	for _, wiface := range wrappedInterfaces {
		wi, err := parseInterface(pkg, wiface)
		if err != nil {
			return "", err
		}
		wrappingIfaces = append(wrappingIfaces, wi)
	}

	// And now begin constructing the file
	f, err := parser.ParseFile(pkg.Fset, "_igen_generated.go", "package "+pkg.Name, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}

	allIfaces := append([]*iface{structSel.iface}, wrappingIfaces...)
	// Add the imports for all interfaces we're going to juggle
	for _, iface := range allIfaces {
		if iface.isCurrentPackage {
			continue
		}
		astutil.AddImport(pkg.Fset, f, iface.pkgPath)
	}

	decls := []ast.Decl{}

	// We need to alias any interfaces that have overlapping names, or else we
	// won't be able to construct structs as we do below.
	wrappingIfaces, aliases := aliasInterfaces(pkg, structSel.iface, wrappingIfaces)
	decls = append(decls, aliases...)

	// generate the function body
	body := &ast.BlockStmt{
		List: []ast.Stmt{},
	}
	// First, the '_, ok0 := s.Iface.(OtherIface)' bit
	for i, iface := range wrappingIfaces {
		body.List = append(body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				ast.NewIdent("_"),
				ast.NewIdent(fmt.Sprintf("ok%d", i)),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.TypeAssertExpr{
					X: &ast.SelectorExpr{
						X:   ast.NewIdent(structSel.receiver),
						Sel: ast.NewIdent(structSel.member.Name()),
					},
					Type: iface.expr(),
				},
			},
		})
	}
	// we now have ok1..n for which interfaces it implements. Now generate the switch statement
	numPerms := 2 << (len(wrappingIfaces) - 1)
	cases := []ast.Stmt{}
	for perm := numPerms - 1; perm >= 0; perm-- {
		// more than 1 iface means we need to wrap them all in a binary expression
		binaryParts := []ast.Expr{}
		// Always include the base interface
		bodyIfaces := []*iface{structSel.iface}
		for i, iface := range wrappingIfaces {
			okNum := fmt.Sprintf("ok%d", i)
			if perm>>i&0x1 == 1 {
				// This one is enabled, so this iface should be used
				bodyIfaces = append(bodyIfaces, iface)
				binaryParts = append(binaryParts, ast.NewIdent(okNum))
			} else {
				binaryParts = append(binaryParts, &ast.UnaryExpr{
					Op: token.NOT,
					X:  ast.NewIdent(okNum),
				})
			}
		}
		// We have the select condition and the ifaces to use in this case.
		// And em all together
		var caseClause ast.Expr
		caseClause = binaryParts[0]
		binaryParts = binaryParts[1:]
		for _, part := range binaryParts {
			caseClause = &ast.BinaryExpr{
				X:  caseClause,
				Op: token.LAND,
				Y:  part,
			}
		}
		// and now we have 'ok0 && !ok1 && ok2 ....' for this perm

		// Now the body
		selectBody := []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					genInterfaceStruct(structSel, bodyIfaces),
				},
			},
		}

		cases = append(cases, &ast.CaseClause{
			List: []ast.Expr{caseClause},
			Body: selectBody,
		})
	}
	// Final case, include the default panic
	cases = append(cases, &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.Ident{
						Name: "panic",
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: `"unreachable"`,
						},
					},
				},
			},
		},
	})

	body.List = append(body.List, &ast.SwitchStmt{
		Body: &ast.BlockStmt{
			List: cases,
		},
	})

	// And now for the function
	wrapFunc := structSel.declareFunction(
		wrapperFuncName,
		body,
	)

	decls = append(decls, wrapFunc)

	// And now generate all the interface implementations
	impldFuncs := map[string]struct{}{}
	for _, iface := range wrappingIfaces {
		for i := 0; i < iface.obj.NumMethods(); i++ {
			method := iface.obj.Method(i)
			if _, ok := impldFuncs[method.Name()]; ok {
				// already impld; hope they're compatible, otherwise we'll fail to compile
				continue
			}
			implFunc := structSel.implementMethod(iface, method)
			impldFuncs[method.Name()] = struct{}{}
			decls = append(decls, implFunc)
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, pkg.Fset, f); err != nil {
		return "", err
	}
	buf.WriteString("\n")
	if err := format.Node(&buf, pkg.Fset, decls); err != nil {
		return "", err
	}
	return buf.String(), err
}

type structSel struct {
	receiver        string
	pointerReceiver bool
	structName      string
	structObj       *types.Struct
	member          *types.TypeName
	iface           *iface
}

func parseStructSel(pkg *packages.Package, s string) (*structSel, error) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("struct selector must contain a space after the receiver name")
	}

	recv := parts[0]
	ptr := strings.HasPrefix(parts[1], "*")
	if ptr {
		// chop off the '*'
		parts[1] = parts[1][1:]
	}
	selParts := strings.Split(parts[1], ".")
	if len(selParts) != 2 {
		return nil, fmt.Errorf("the struct selector must be of the form 'structName.Field', but %v did not have a dot", parts[1])
	}

	structName, memberName := selParts[0], selParts[1]

	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		return nil, fmt.Errorf("Could not find any struct named %q in package %q", structName, pkg.Name)
	}

	memberObj, _, _ := types.LookupFieldOrMethod(obj.Type(), false, obj.Pkg(), memberName)
	if memberObj == nil {
		return nil, fmt.Errorf("struct %q in pkg %q had no member %q", structName, pkg.Name, memberName)
	}

	if !types.IsInterface(memberObj.Type()) {
		return nil, fmt.Errorf("'%v.%v' in pkg %q was not an interface", structName, memberName, pkg.Name)
	}

	return &structSel{
		receiver:        recv,
		pointerReceiver: ptr,
		structName:      structName,
		structObj:       obj.Type().(*types.Named).Underlying().(*types.Struct),
		member:          memberObj.(*types.Var).Type().(*types.Named).Obj(),
		iface: &iface{
			pkgName:          memberObj.(*types.Var).Type().(*types.Named).Obj().Pkg().Name(),
			pkgPath:          memberObj.(*types.Var).Type().(*types.Named).Obj().Pkg().Path(),
			isCurrentPackage: false, // this may be wrong :(
			name:             memberObj.Name(),
			obj:              memberObj.Type().Underlying().(*types.Interface),
		},
	}, nil
}

func (s *structSel) declareFunction(name string, body *ast.BlockStmt) *ast.FuncDecl {
	var recv ast.Expr
	if s.pointerReceiver {
		recv = &ast.StarExpr{
			X: ast.NewIdent(s.structName),
		}
	} else {
		recv = ast.NewIdent(s.structName)
	}

	return &ast.FuncDecl{
		Name: ast.NewIdent(name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: ast.NewIdent(s.member.Type().String())}},
			},
		},
		Recv: &ast.FieldList{
			List: []*ast.Field{{
				Names: []*ast.Ident{{
					Name: s.receiver,
					Obj: &ast.Object{
						Kind: ast.Var,
						Name: s.receiver,
						// Decl: &ast.FuncDecl{},
					},
				}},
				Type: recv,
			}},
		},
		Body: body,
	}
}

func (s *structSel) implementMethod(iface *iface, method *types.Func) *ast.FuncDecl {
	var recv ast.Expr
	if s.pointerReceiver {
		recv = &ast.StarExpr{
			X: ast.NewIdent(s.structName),
		}
	} else {
		recv = ast.NewIdent(s.structName)
	}

	// TODO: variadic functions
	sig := method.Type().(*types.Signature)
	params := sig.Params()
	args := []*ast.Field{}
	callArgs := []ast.Expr{}
	for i := 0; i < params.Len(); i++ {
		arg := params.At(i)
		args = append(args, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(arg.Name())},
			Type:  ast.NewIdent(arg.Type().String()),
		})
		// TODO: '...' calling
		callArgs = append(callArgs, ast.NewIdent(arg.Name()))
	}

	results := []*ast.Field{}
	ret := sig.Results()
	for i := 0; i < ret.Len(); i++ {
		arg := ret.At(i)
		results = append(results, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(arg.Name())},
			Type:  ast.NewIdent(arg.Type().String()),
		})
	}

	body := []ast.Stmt{
		&ast.ReturnStmt{
			Results: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.TypeAssertExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent(s.receiver),
								Sel: ast.NewIdent(s.member.Name()),
							},
							Type: iface.expr(),
						},
						Sel: ast.NewIdent(method.Name()),
					},
					Args: callArgs,
				},
			},
		},
	}

	return &ast.FuncDecl{
		Name: ast.NewIdent(method.Name()),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: args},
			Results: &ast.FieldList{List: results},
		},
		Recv: &ast.FieldList{
			List: []*ast.Field{{
				Names: []*ast.Ident{{
					Name: s.receiver,
					Obj: &ast.Object{
						Kind: ast.Var,
						Name: s.receiver,
					},
				}},
				Type: recv,
			}},
		},
		Body: &ast.BlockStmt{
			List: body,
		},
	}
}

type iface struct {
	pkgPath          string
	pkgName          string
	isCurrentPackage bool
	name             string
	obj              *types.Interface
}

func parseInterface(pkg *packages.Package, s string) (*iface, error) {
	// 'io.Reader' for example -> [io, Reader]
	parts := strings.SplitN(s, ".", 2)
	var pkgName, ifaceName string
	if len(parts) == 2 {
		pkgName, ifaceName = parts[0], parts[1]
	} else {
		ifaceName = parts[0]
	}
	// Same pkg case
	ifacePkg := pkg
	if pkgName != "" {
		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.NeedTypes | packages.NeedName | packages.NeedSyntax | packages.NeedImports,
		}, pkgName)
		if err != nil {
			return nil, fmt.Errorf("error loading package %q: %w", pkgName, err)
		}
		ifacePkg = pkgs[0]
	}

	obj := ifacePkg.Types.Scope().Lookup(ifaceName)
	if obj == nil {
		return nil, fmt.Errorf("no interface named %q in package %q", ifaceName, pkg.Name)
	}

	if !types.IsInterface(obj.Type()) {
		return nil, fmt.Errorf("%q in package %q is not an interface", ifaceName, pkgName)
	}

	return &iface{
		ifacePkg.PkgPath,
		ifacePkg.Name,
		pkgName == "",
		ifaceName,
		obj.Type().Underlying().(*types.Interface),
	}, nil
}

func (i *iface) expr() ast.Expr {
	if i.isCurrentPackage {
		return ast.NewIdent(i.name)
	}
	return &ast.SelectorExpr{
		X:   ast.NewIdent(i.pkgName),
		Sel: ast.NewIdent(i.name),
	}
}

func genInterfaceStruct(s *structSel, ifaces []*iface) *ast.CompositeLit {
	fields := []*ast.Field{}
	elts := []ast.Expr{}
	for _, iface := range ifaces {
		fields = append(fields, &ast.Field{
			Type: iface.expr(),
		})

		elts = append(elts, ast.NewIdent(s.receiver))
	}
	return &ast.CompositeLit{
		Type: &ast.StructType{
			Fields: &ast.FieldList{
				List: fields,
			},
		},
		Elts: elts,
	}
}

func aliasInterfaces(pkg *packages.Package, s *iface, ifaces []*iface) ([]*iface, []ast.Decl) {
	ret := make([]*iface, 0, len(ifaces))
	used := map[string]struct{}{s.name: {}}

	decls := []ast.Decl{}

	for _, ifc := range ifaces {
		if _, taken := used[ifc.name]; !taken {
			ret = append(ret, ifc)
			used[ifc.name] = struct{}{}
			continue
		}
		// Otherwise, create an alias
		name := "igenIfaceAlias"
		for suffix := 0; true; suffix++ {
			if _, taken := used[name+fmt.Sprintf("%d", suffix)]; taken {
				continue
			}
			name += fmt.Sprintf("%d", suffix)
			break
		}
		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: ast.NewIdent(name),
					Type: &ast.InterfaceType{
						Methods: &ast.FieldList{
							List: []*ast.Field{
								{
									Type: ifc.expr(),
								},
							},
						},
					},
				},
			},
		})

		used[name] = struct{}{}
		ret = append(ret, &iface{
			pkgName:          pkg.Name,
			pkgPath:          pkg.PkgPath,
			isCurrentPackage: true,
			name:             name,
			obj:              ifc.obj,
		})
	}

	return ret, decls
}