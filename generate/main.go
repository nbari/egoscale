package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/exoscale/egoscale"
)

var cmd = flag.String("cmd", "", "")

func main() {
	// GOPATH
	//gopath, _ := os.LookupEnv("GOPATH")
	// GOFILE
	//gofile, _ := os.LookupEnv("GOFILE")
	// GOPACKAGE
	gopackage, _ := os.LookupEnv("GOPACKAGE")
	// GOLINE
	//_goline, _ := os.LookupEnv("GOLINE")
	//__goline, _ := strconv.ParseInt(_goline, 10, 32)
	//goline := int(__goline)

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "provide json file!")
		os.Exit(1)
	}

	var source = flag.Arg(0)

	sourceFile, _ := os.Open(source)
	decoder := json.NewDecoder(sourceFile)
	apis := new(egoscale.ListAPIsResponse)
	if err := decoder.Decode(&apis); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(2)
	}

	fset := token.NewFileSet()
	astFiles := make([]*ast.File, 0)
	files, err := filepath.Glob("*.go")
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(2)
		}
		astFiles = append(astFiles, f)
	}

	info := types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}

	conf := types.Config{
		Importer: importer.For("source", nil),
	}

	_, err = conf.Check(gopackage, fset, astFiles, &info)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(2)
	}

	var s *types.Struct
	for id, obj := range info.Defs {
		if obj == nil || !obj.Exported() {
			continue
		}

		typ := obj.Type().Underlying()

		pos := fset.Position(id.Pos())

		switch typ.(type) {
		case *types.Struct:
			if strings.ToLower(obj.Name()) == strings.ToLower(*cmd) {
				fmt.Printf("%s: %s\n", pos, obj.Name())
				s = typ.(*types.Struct)
			}
		}
	}

	if s == nil {
		fmt.Fprintf(os.Stderr, "Definition %q not found. Are you in right place?\n", *cmd)
		os.Exit(2)
	}

	type fieldInfo struct {
		Var       *types.Var
		OmitEmpty bool
		Doc       string
	}

	re := regexp.MustCompile(`\bjson:"(?P<name>[^,"]+)(?P<omit>,omitempty)?"`)
	reDoc := regexp.MustCompile(`\bdoc:"(?P<doc>[^"]+)"`)
	for _, a := range apis.API {
		if a.Name == *cmd {
			// name to field
			fields := make(map[string]fieldInfo)
			for i := 0; i < s.NumFields(); i++ {
				f := s.Field(i)
				if !f.IsField() || !f.Exported() {
					continue
				}

				tag := s.Tag(i)
				match := re.FindStringSubmatch(tag)
				if len(match) == 0 {
					fmt.Fprintf(os.Stderr, "Field error: no json annotation found for %s.", f.Name())
					continue
				}
				name := match[1]
				omitempty := len(match) == 3 && match[2] == ",omitempty"

				doc := ""
				match = reDoc.FindStringSubmatch(tag)
				if len(match) == 2 {
					doc = match[1]
				}

				//fmt.Printf("tag: %q %v\n", match[1], omitempty)
				fields[name] = fieldInfo{
					Var:       f,
					OmitEmpty: omitempty,
					Doc:       doc,
				}
			}
			//fmt.Printf("fields %v\n", fields)

			for _, p := range a.Params {
				field, ok := fields[p.Name]

				if !ok {
					fmt.Fprintf(os.Stderr, "Field missing: expected to find %q\n", p.Name)
					continue
				}
				delete(fields, p.Name)

				typename := field.Var.Type().String()
				expected := ""
				switch p.Type {
				case "integer":
					if typename != "int" {
						expected = "int"
					}
				case "long":
					if typename != "int64" {
						expected = "int64"
					}
				case "boolean":
					if typename != "bool" && typename != "*bool" {
						expected = "bool"
					}
				case "string":
				case "uuid":
					if typename != "string" {
						expected = "string"
					}
				case "map":
					if !strings.HasPrefix(typename, "[]") {
						expected = "array"
					}
				default:
					fmt.Fprintf(os.Stderr, "Field %q: unknown type: %q <=> %s\n", p.Name, p.Type, field.Var.Type().String())
				}

				if expected != "" {
					fmt.Fprintf(os.Stderr, "Field %q expected to be an array, got %s\n", p.Name, typename)
				}

				if field.Doc != p.Description {
					fmt.Fprintf(os.Stderr, "Field %q: use `doc:%q`\n", p.Name, p.Description)
				}
			}

			for name := range fields {
				fmt.Fprintf(os.Stderr, "Field %q was defined but doesn't exist\n", name)
			}

			os.Exit(0)
		}
	}

	os.Exit(3)
}
