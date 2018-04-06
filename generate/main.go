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
	"strconv"

	"github.com/exoscale/egoscale"
)

var cmd = flag.String("cmd", "", "")

func main() {
	// GOPATH
	//gopath, _ := os.LookupEnv("GOPATH")
	// GOFILE
	gofile, _ := os.LookupEnv("GOFILE")
	// GOPACKAGE
	gopackage, _ := os.LookupEnv("GOPACKAGE")
	// GOLINE
	_goline, _ := os.LookupEnv("GOLINE")
	__goline, _ := strconv.ParseInt(_goline, 10, 32)
	goline := int(__goline)

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "provide json file!")
		os.Exit(1)
	}

	var source = flag.Arg(0)
	fmt.Printf("%s cmd=%s\n", source, *cmd)

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

		pos := fset.Position(id.Pos())
		if gofile != pos.Filename || goline-pos.Line != -1 {
			continue
		}

		typ := obj.Type().Underlying()
		switch typ.(type) {
		case *types.Struct:
			s = typ.(*types.Struct)
		}
	}

	for _, a := range apis.API {
		if a.Name == *cmd {
			// TODO
			fmt.Println("Make the match between")
			fmt.Printf("json: %#v\n\n", a)
			fmt.Printf("golang: %#v\n", s)
			os.Exit(0)
		}
	}
}
