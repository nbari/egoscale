package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/go-ini/ini"
	flag "github.com/spf13/pflag"
)

func main() {
	// XXX having all the methods! ~> pkgreflect
	methods := []interface{}{
		&egoscale.ListVirtualMachines{},
		&egoscale.ListAPIs{},
		&egoscale.ListZones{},
		&egoscale.DeployVirtualMachine{},
	}

	if len(os.Args) <= 1 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  %s command\n\n", os.Args[0])

		fmt.Fprintln(os.Stderr, "Available commands:")
		fmt.Fprintln(os.Stderr, "")
		for _, m := range methods {
			name := m.(egoscale.Command).APIName()
			fmt.Fprintf(os.Stderr, "  %s\n", name)
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "no commands found")
		os.Exit(1)
	}

	command := os.Args[1]

	var method interface{}
	for _, m := range methods {
		if c, ok := m.(egoscale.Command); ok {
			if m.(egoscale.Command).APIName() == command {
				method = c
			}
		} else {
			log.Panicf("%+v is not a command!", m)
		}
	}

	if method == nil {
		// XXX do a "did you mean?"
		log.Fatalf("%s is not a known command", command)
	}

	flagset := flag.NewFlagSet(command, flag.ExitOnError)
	// global setting
	var region string
	flagset.StringVarP(&region, "region", "r", "cloudstack", "cloudstack.ini section name")

	val := reflect.ValueOf(method)
	// we've go a pointer
	val = val.Elem()
	if err := populateVars(flagset, val); err != nil {
		panic(err)
	}
	flagset.Parse(os.Args[2:])

	client, _ := buildClient(region)

	// Request
	//out, _ := json.MarshalIndent(&method, "", "  ")
	//fmt.Println(string(out))
	resp, err := client.Request(method.(egoscale.Command))
	if err != nil {
		out, _ := json.MarshalIndent(err, "", "  ")
		log.Panicf(string(out))
	}
	out, _ := json.MarshalIndent(&resp, "", "  ")
	fmt.Println(string(out))
}

func buildClient(region string) (*egoscale.Client, error) {
	usr, _ := user.Current()
	localConfig, _ := filepath.Abs("cloudstack.ini")
	inis := []string{
		localConfig,
		filepath.Join(usr.HomeDir, ".cloudstack.ini"),
	}
	config := ""
	for _, i := range inis {
		if _, err := os.Stat(i); err != nil {
			continue
		}
		config = i
		break
	}

	if config == "" {
		log.Panicf("Config file not found within: %s", strings.Join(inis, ", "))
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, config)
	if err != nil {
		log.Panic(err)
	}

	section, err := cfg.GetSection(region)
	if err != nil {
		log.Panicf("Section %q not found in the config file %s", region, config)
	}
	endpoint, _ := section.GetKey("endpoint")
	key, _ := section.GetKey("key")
	secret, _ := section.GetKey("secret")

	client := egoscale.NewClient(endpoint.String(), key.String(), secret.String())

	return client, nil
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

		if doc, ok := field.Tag.Lookup("doc"); ok {
			if description != "" {
				description = fmt.Sprintf("[%s] %s", description, doc)
			} else {
				description = doc
			}
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
		case reflect.Slice:
			switch field.Type.Elem().Kind() {
			case reflect.Uint8:
				ip := addr.(*net.IP)
				if *ip == nil || (*ip).Equal(net.IPv4zero) || (*ip).Equal(net.IPv6zero) {
					flagset.IPVar(ip, argName, nil, description)
				}
			case reflect.String:
				flagset.StringSliceVar(addr.(*[]string), argName, nil, description)
			default:
				log.Printf("[SKIP] Type of %s is not supported!", field.Name)
			}
		case reflect.Map:
			log.Printf("[SKIP] Type map for %s is not supported!", field.Name)
		default:
			log.Printf("[SKIP] Type of %s is not supported!", field.Name)
		}
	}
	return nil
}
