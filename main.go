// Command cmtstringer is a tool that help to generate method `func (t T) String() string`,
// which satisfy the fmt.Stringer interface, for given type name of a constant.
// Returned value is the comment text of the constant.
//
// Install
//
// To install command cmtstringer, run command
//
//	go get github.com/lazada/cmtstringer
//
// Usage
//
// For example, given this file
//
// 	package http
//
// 	//go:generate cmtstringer -type StatusCode
//
// 	type StatusCode int
//
// 	const (
// 		// StatusBadRequest Bad Request
// 		StatusBadRequest StatusCode = 400
// 		// StatusNotFound Not Found
// 		StatusNotFound StatusCode = 404
// 	)
//
// run command
//
// 	go generate
//
// then you will get file `statuscode_string_gen.go`
//
// 	package http
//
// 	// This file is generated by command cmtstringer.
// 	// DO NOT EDIT IT.
//
// 	// String returns comment of const type StatusCode
// 	func (s StatusCode) String() string {
// 		switch s {
// 		case StatusBadRequest:
// 			return "Bad Request"
// 		case StatusNotFound:
// 			return "Not Found"
// 		default:
// 			return "Unknown"
// 		}
// 	}
//
package main // import "github.com/lazada/cmtstringer"

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	typeName = flag.String("type", "", "type name of const; must be set.")
	output   = flag.String("output", "", "output file name; default srcdir/<type>_string_gen.go")
)

const (
	fileTemplateStr = `package {{.PackageName}}

// This file is generated by command cmtstringer.
// DO NOT EDIT IT.

// String returns comment of const type {{.TypeName}}
func ({{.Reciever}} {{.TypeName}}) String() string {
	switch {{.Reciever}} {
	{{range .Consts}}case {{.Name}}:
		return {{printf "%q" .Msg}}
	{{end}}default:
		return "Unknown"
	}
}
`
)

var (
	fileTemplate = template.Must(template.New("fileTemplate").Parse(fileTemplateStr))
)

// constValue represents information of an constant
type constValue struct {
	Name string
	Msg  string
}

// Usage is a replacement usage function for the flags package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprint(os.Stderr, "\tcmtstringer [options] -type T [directory]\n")
	fmt.Fprint(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

func init() {
	log.SetFlags(0)
	log.SetPrefix("cmtstringer: ")

	flag.Usage = Usage
	flag.Parse()
}

func main() {
	if *typeName == "" {
		flag.Usage()
		os.Exit(2)
	}

	args := flag.Args()
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	dir := args[0]
	if !isDirectory(dir) {
		flag.Usage()
		os.Exit(2)
	}

	parseDir(dir)
}

func parseDir(dir string) {
	fset := token.NewFileSet() // positions are relative to fset
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	numPkgs := len(pkgs)
	for pkgName, pkg := range pkgs {
		checkPackages(dir, fset, pkg)

		values := parsePackage(pkg)

		if len(values) == 0 {
			continue
		}

		tmplData := struct {
			PackageName string
			TypeName    string
			Reciever    string
			Consts      []constValue
		}{
			PackageName: pkgName,
			TypeName:    *typeName,
			Reciever:    strings.ToLower(string((*typeName)[0])),
			Consts:      values,
		}

		outputName := *output
		if outputName == "" {
			baseName := fmt.Sprintf("%s_string_gen.go", *typeName)
			outputName = filepath.Join(dir, strings.ToLower(baseName))
		}

		if numPkgs > 1 {
			outputName = fmt.Sprintf("%s_%s", pkgName, outputName)
		}

		genfile(outputName, fileTemplate, tmplData)
	}
}

func parsePackage(pkg *ast.Package) []constValue {
	values := []constValue{}
	for _, f := range pkg.Files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			if gd.Tok != token.CONST {
				continue
			}

			var typ string
			for _, s := range gd.Specs {
				vs := s.(*ast.ValueSpec)

				if vs.Type == nil && len(vs.Values) > 0 {
					// "X = 1". With no type but a value, the constant is untyped.
					// Skip this vspec and reset the remembered type.
					typ = ""
					continue
				}

				if vs.Type != nil {
					// "X T". We have a type. Remember it.
					ident, ok := vs.Type.(*ast.Ident)
					if !ok {
						continue
					}
					typ = ident.Name
				}

				if typ != *typeName {
					continue
				}

				for i := range vs.Names {
					if vs.Names[i] == nil {
						continue
					}

					if vs.Names[i].Name == "_" || !vs.Names[i].IsExported() {
						continue
					}

					var constName = vs.Names[i].String()
					var message string
					if vs.Doc != nil {
						comment := vs.Doc.Text()
						if strings.HasPrefix(comment, constName) {
							nlReplacer := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")
							message = nlReplacer.Replace(comment)
							message = strings.TrimPrefix(message, constName)
							message = strings.TrimSpace(message)
						}
					}

					cv := constValue{
						Name: constName,
						Msg:  message,
					}

					values = append(values, cv)
				}
			}
		}
	}

	return values
}

func genfile(fileName string, fileTemplate *template.Template, tmplData interface{}) {
	buf := bytes.Buffer{}
	if err := fileTemplate.Execute(&buf, tmplData); err != nil {
		log.Fatal(err)
	}

	fmtSource, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(fileName, fmtSource, 0664)
	if err != nil {
		log.Fatal(err)
	}
}

// isDirectory reports whether the named file is a directory.
func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}
	return info.IsDir()
}

func checkPackages(dir string, fset *token.FileSet, p *ast.Package) {
	defs := make(map[*ast.Ident]types.Object)
	config := types.Config{Importer: importer.Default(), FakeImportC: true}
	info := &types.Info{Defs: defs}
	files := make([]*ast.File, 0, len(p.Files))
	for _, f := range p.Files {
		files = append(files, f)
	}
	_, err := config.Check(dir, fset, files, info)
	if err != nil {
		log.Fatalf("checking package: %v", err)
	}
}
