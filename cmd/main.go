package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/exoscale/egoscale"
)

func main() {
	// XXX having all the methods!
	methods := []interface{}{
		&egoscale.ListVirtualMachines{},
		&egoscale.ListZones{},
	}

	if len(os.Args) <= 1 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintf(os.Stderr, "\n  %s command\n\n", os.Args[0])

		fmt.Fprintln(os.Stderr, "Available commands:\n")
		for _, m := range methods {
			fmt.Fprintf(os.Stderr, "  %s\n", m.(egoscale.Command).APIName())
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "no command found")
		os.Exit(1)
	}

	command := os.Args[1]

	isAsync := false
	var method interface{}
	for _, m := range methods {
		if c, ok := m.(egoscale.Command); ok {
			if c.APIName() == command {
				method = c
			}
		} else {
			log.Panicf("%+v is not a command!", m)
		}
		if _, ok := m.(egoscale.AsyncCommand); ok {
			isAsync = ok
		}
	}

	if method == nil {
		// XXX do a "did you mean?"
		log.Fatalf("%s is not a known command", command)
	}

	flagset := flag.NewFlagSet(command, flag.ExitOnError)
	val := reflect.ValueOf(method)
	// we've go a pointer
	val = val.Elem()
	if err := populateVars(flagset, val); err != nil {
		panic(err)
	}
	flagset.Parse(os.Args[2:])

	out, _ := json.MarshalIndent(&method, "", "  ")
	fmt.Printf("%s (async=%v)\n", command, isAsync)
	fmt.Println(string(out))
}

func populateVars(flagset *flag.FlagSet, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("struct was expected")
	}
	ty := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := ty.Field(i)

		// XXX refactor with request.go
		var argName string
		required := false
		if json, ok := field.Tag.Lookup("json"); ok {
			tags := strings.Split(json, ",")
			argName = tags[0]
			required = true
			for _, tag := range tags {
				if tag == "omitempty" {
					required = false
				}
			}
			if argName == "" || argName == "omitempty" {
				argName = strings.ToLower(field.Name)
			}
		}

		description := ""
		if required {
			description = "required"
		}

		val := value.Field(i)
		addr := val.Addr().Interface()
		switch val.Kind() {
		case reflect.Bool:
			flagset.BoolVar(addr.(*bool), argName, false, description)
		case reflect.Int:
			flagset.IntVar(addr.(*int), argName, 0, description)
		case reflect.Int64:
			flagset.Int64Var(addr.(*int64), argName, 0, description)
		case reflect.Uint:
			flagset.UintVar(addr.(*uint), argName, 0, description)
		case reflect.Uint64:
			flagset.Uint64Var(addr.(*uint64), argName, 0, description)
		case reflect.Float64:
			flagset.Float64Var(addr.(*float64), argName, 0., description)
		case reflect.String:
			flagset.StringVar(addr.(*string), argName, "", description)
		case reflect.Slice, reflect.Map:
			// pass
		default:
			log.Printf("[SKIP] Type %s is not supported!", field.Name)
		}
	}
	return nil
}
