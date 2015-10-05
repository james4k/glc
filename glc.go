package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
)

func main() {
	c := &context{}
	c.fset = token.NewFileSet()

	f, err := parser.ParseFile(
		c.fset,
		"test/var.glc",
		nil,
		parser.ParseComments|parser.AllErrors,
	)
	if err != nil {
		log.Println(err)
	}

	var imports []*ast.GenDecl
	var consts []*ast.GenDecl
	var types []*ast.GenDecl
	var vars []*ast.GenDecl
	var funcs []*ast.FuncDecl

	for _, d := range f.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			switch decl.Tok {
			case token.IMPORT:
				imports = append(imports, decl)
			case token.CONST:
				consts = append(consts, decl)
			case token.TYPE:
				types = append(types, decl)
			case token.VAR:
				vars = append(vars, decl)
			}
		case *ast.FuncDecl:
			funcs = append(funcs, decl)
		default:
			log.Fatalln("unknown decl")
		}
	}

	for _, v := range vars {
		for _, s := range v.Specs {
			value := s.(*ast.ValueSpec)
			for i := range value.Names {
				fmt.Printf("%v %s = %s;\n", value.Type, value.Names[i], value.Values[i].(*ast.BasicLit).Value)
			}
		}
	}

	// function declarations
	for _, f := range funcs {
		var returnType string
		var paramTypes []string
		var paramNames []string
		name := f.Name.Name
		//for _, r := range f.Recv.List {
		//	fmt.Printf("\n", f.Recv)
		//}
		for _, p := range f.Type.Params.List {
			typ := p.Type.(*ast.Ident).Name
			for _, n := range p.Names {
				paramTypes = append(paramTypes, typ)
				paramNames = append(paramNames, n.Name)
			}
		}
		if f.Type.Results != nil {
			for _, p := range f.Type.Results.List {
				returnType = p.Type.(*ast.Ident).Name
				/*
					for _, n := range p.Names {
						fmt.Printf("  %v\n", n.Name)
					}
				*/
				break
			}
		}
		if returnType == "" {
			returnType = "void"
		}
		fmt.Printf("%s %s(", returnType, name)
		for i := range paramTypes {
			delim := ""
			if i != 0 {
				delim = ","
			}
			fmt.Printf("%s %s%s", paramTypes[i], paramNames[i], delim)
		}
		fmt.Println(") {")
		genBlock(c, f.Body)
		fmt.Println("}")
	}

	// function definitions
}

type context struct {
	fset *token.FileSet
}

func parseError(c *context, n ast.Node, s string) {
	log.Printf("%v: s", c.fset.Position(n.Pos()), s)
}

func genBlock(c *context, block *ast.BlockStmt) {
	for _, stmt := range block.List {
		genStmt(c, stmt)
	}
}

func genStmt(c *context, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		fmt.Println("{")
		genBlock(c, s)
		fmt.Println("}")
	case *ast.DeclStmt:
		gendecl := s.Decl.(*ast.GenDecl)
		switch gendecl.Tok {
		case token.CONST:
			parseError(c, s, "const not currently supported")
		case token.TYPE:
			parseError(c, s, "local types not currently supported")
		case token.VAR:
			for _, s := range gendecl.Specs {
				value := s.(*ast.ValueSpec)
				switch {
				case len(value.Values) == 0:
					for i := range value.Names {
						fmt.Printf(
							"%v %s;\n",
							value.Type,
							value.Names[i],
						)
					}
				case len(value.Values) != len(value.Names):
					parseError(c, gendecl, "assignment count mismatch")
				default:
					for i := range value.Names {
						fmt.Printf(
							"%v %s = %s;\n",
							value.Type,
							value.Names[i],
							value.Values[i].(*ast.BasicLit).Value,
						)
					}
				}
			}
		}
	case *ast.ExprStmt:
		genExpr(c, s.X)
		fmt.Print(";\n")
	case *ast.IfStmt:
		fmt.Print("if (")
		genExpr(c, s.Cond)
		fmt.Print(") {\n")
		genBlock(c, s.Body)
		if s.Else != nil {
			fmt.Print("} else ")
			genStmt(c, s.Else)
		} else {
			fmt.Print("}\n")
		}
	case *ast.ReturnStmt:
		switch len(s.Results) {
		case 0:
			fmt.Println("return;")
		case 1:
			fmt.Printf("return ")
			genExpr(c, s.Results[0])
			fmt.Println(";")
		default:
			parseError(c, s, "multiple return values not currently supported")
		}
	case *ast.AssignStmt:
		if len(s.Lhs) > 1 || len(s.Rhs) > 1 {
			parseError(c, s, "multiple assignment not currently supported")
		} else {
			genExpr(c, s.Lhs[0])
			fmt.Print(" = ")
			genExpr(c, s.Rhs[0])
			fmt.Println(";")
		}
	default:
		fmt.Printf("%#v\n", stmt)
	}
}

func genExpr(c *context, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT, token.FLOAT, token.CHAR, token.STRING:
			fmt.Print(e.Value)
		case token.IMAG:
			parseError(c, e, "complex numbers not currently supported")
		}
	case *ast.Ident:
		fmt.Print(e.Name)
	case *ast.BinaryExpr:
		genExpr(c, e.X)
		fmt.Printf(" %v ", e.Op)
		genExpr(c, e.Y)
	case *ast.CallExpr:
		genExpr(c, e.Fun)
		fmt.Print("(")
		for _, a := range e.Args {
			genExpr(c, a)
		}
		fmt.Print(")")
	default:
		log.Printf("%#v\n", e)
	}
}
