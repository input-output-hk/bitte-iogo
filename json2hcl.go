package main

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/nomad/api"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type Json2HclCmd struct {
	Output string `arg:"-o" help:"write output to this file (- for stdout)" placeholder:"FILE"`
	Input  string `arg:"-i" help:"read JSON from this file (- for stdin)" placeholder:"FILE"`
}

func runJson2Hcl(args *Json2HclCmd) error {
	read, err := openInput(args.Input)
	if err != nil {
		return err
	}

	input, err := io.ReadAll(read)
	if err != nil {
		return err
	}

	wrapper := &JobWrapper{}

	err = json.Unmarshal(input, wrapper)
	if err != nil {
		return err
	}

	file, err := any2hcl("job", wrapper.Job)
	if err != nil {
		return errors.WithMessage(err, "Trying to transform Job to HCL")
	}

	write, err := openOutput(args.Output)
	if err != nil {
		return err
	}

	_, err = file.WriteTo(write)
	return err
}

func any2hcl(key string, any interface{}) (*hclwrite.File, error) {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	rv := reflect.ValueOf(any)
	if rv.Kind() == reflect.Ptr {
		return f, convert(body, key, rv.Elem().Interface())
	} else {
		return f, convert(body, key, any)
	}
}

type structField struct {
	Name     string
	Block    bool
	Optional bool
	Label    bool
	Value    interface{}
}

func (s *structField) Cty() cty.Value {
	switch v := s.Value.(type) {
	case nil:
		return cty.NilVal
	case string:
		if v == "" && s.Optional {
			return cty.NilVal
		} else {
			return cty.StringVal(v)
		}
	case *string:
		if v == nil {
			return cty.NilVal
		} else {
			return cty.StringVal(*v)
		}
	case int8:
		return cty.NumberIntVal(int64(v))
	case *int8:
		if v == nil {
			return cty.NilVal
		} else {
			return cty.NumberIntVal(int64(*v))
		}
	case uint64:
		return cty.NumberIntVal(int64(v))
	case *uint64:
		if v == nil {
			return cty.NilVal
		} else {
			return cty.NumberIntVal(int64(*v))
		}
	case int:
		if v == 0 && s.Optional {
			return cty.NilVal
		} else {
			return cty.NumberIntVal(int64(v))
		}
	case *int:
		if v == nil {
			return cty.NilVal
		} else {
			return cty.NumberIntVal(int64(*v))
		}
	case bool:
		if !v && s.Optional {
			return cty.NilVal
		} else {
			return cty.BoolVal(v)
		}
	case *bool:
		if v == nil {
			return cty.NilVal
		} else {
			return cty.BoolVal(*v)
		}
	case time.Duration:
		if v == 0 && s.Optional {
			return cty.NilVal
		} else {
			return cty.StringVal(v.String())
		}
	case *time.Duration:
		if v == nil {
			return cty.NilVal
		} else {
			return cty.StringVal(v.String())
		}
	case []string:
		if len(v) == 0 {
			if s.Optional {
				return cty.NilVal
			} else {
				return cty.ListValEmpty(cty.String)
			}
		} else {
			converted := []cty.Value{}
			for _, value := range v {
				converted = append(converted, cty.StringVal(value))
			}
			return cty.ListVal(converted)
		}
	default:
		panic(fmt.Sprintf("Unknown type for: %t, %#t %#v", v, v, v))
	}
}

func convert(parent *hclwrite.Body, key string, obj interface{}) error {
	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)

	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	// skip everything without value
	switch objValue.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Struct:
		if obj == nil {
			return nil
		}
	case reflect.Array, reflect.Slice, reflect.Map:
		if objValue.Len() == 0 {
			return nil
		}
	default:
		fmt.Println("objType.Kind:", objType.Kind(), "objValue.kind:", objValue.Kind())
	}

	switch objValue.Kind() {
	case reflect.Map:
		if len(parent.Attributes()) > 0 || len(parent.Blocks()) > 0 {
			parent.AppendNewline()
		}

		switch t := obj.(type) {
		case map[string]*api.VolumeRequest:
			for mapKey, mapValue := range t {
				body := parent.AppendNewBlock(key, []string{mapKey}).Body()
				convert(body, "", *mapValue)
			}
		case map[string]*api.ConsulGatewayBindAddress:
			for mapKey, mapValue := range t {
				body := parent.AppendNewBlock(key, []string{mapKey}).Body()
				convert(body, "", *mapValue)
			}
		case map[string]string, map[string]interface{}:
			body := parent.AppendNewBlock(key, []string{}).Body()

			keyValues := objValue.MapKeys()
			keys := []string{}
			for _, keyValue := range keyValues {
				keys = append(keys, keyValue.String())
			}

			sort.Strings(keys)

			for _, mapKey := range keys {
				mapValue := objValue.MapIndex(reflect.ValueOf(mapKey))

				v, ok := convertValue(mapValue.Interface())
				if !ok {
					return fmt.Errorf("Couldn't convert value: %#v", mapValue.Interface())
				}

				setCty(body, mapKey, v)
			}

		default:
			return fmt.Errorf("Unknown type: %T", t)
		}

	case reflect.Struct:
		if len(parent.Attributes()) > 0 || len(parent.Blocks()) > 0 {
			parent.AppendNewline()
		}

		structFields := []*structField{}
		var blockLabel string

		for i := 0; i < objType.NumField(); i++ {
			field := objType.Field(i)
			tag := field.Tag.Get("hcl")
			if tag == "" {
				continue
			}

			tags := strings.Split(tag, ",")
			var name string
			block := false
			optional := false
			label := false

			for j, elem := range tags {
				if j == 0 {
					name = elem
				} else if elem == "block" {
					block = true
				} else if elem == "optional" {
					optional = true
				} else if elem == "label" {
					label = true
				} else {
					logger.Fatalf("Unknown hcl tag: %s", elem)
				}
			}

			fieldValue := objValue.Field(i)

			if label {
				if fieldValue.Kind() == reflect.Ptr {
					blockLabel = fieldValue.Elem().String()
				} else {
					blockLabel = fieldValue.String()
				}
			} else if name != "" {
				var structValue interface{}

				if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() {
					structValue = fieldValue.Elem().Interface()
				} else {
					structValue = fieldValue.Interface()
				}

				// fmt.Printf("block:%-5v optional:%-5v label:%-5v name:%-20s value:%#v\n", block, optional, label, name, structValue)
				structFields = append(structFields, &structField{name, block, optional, label, structValue})
			}

			// why is job special?
			if key == "job" && name == "name" && blockLabel == "" {
				if fieldValue.Kind() == reflect.Ptr {
					blockLabel = fieldValue.Elem().String()
				} else {
					blockLabel = fieldValue.String()
				}
			}
		}

		var body *hclwrite.Body

		if key == "" {
			body = parent
		} else {
			if blockLabel == "" {
				body = parent.AppendNewBlock(key, []string{}).Body()
			} else {
				body = parent.AppendNewBlock(key, []string{blockLabel}).Body()
			}
		}

		for _, field := range structFields {
			switch {
			case field.Block && field.Optional:
				fmt.Printf("%15s %5v %5v %5v: %#v\n", field.Name, field.Block, field.Optional, field.Label, field.Value)
			case field.Block && !field.Optional:
				// fmt.Printf("%15s %5v %5v %5v: %#v\n", field.Name, field.Block, field.Optional, field.Label, field.Value)
				err := convert(body, field.Name, field.Value)
				if err != nil {
					return err
				}
			case !field.Block && field.Optional:
				if key == "job" && field.Name == "name" {
					continue
				}
				if c := field.Cty(); c != cty.NilVal {
					setCty(body, field.Name, c)
				}
			case !field.Block && !field.Optional:
				fmt.Printf("%15s %5v %5v %5v: %#v\n", field.Name, field.Block, field.Optional, field.Label, field.Value)
			}
		}
	case reflect.Slice:
		// body := parent.AppendNewBlock(key, []string{}).Body()
		for i := 0; i < objValue.Len(); i++ {
			elem := objValue.Index(i)
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			err := convert(parent, key, elem.Interface())
			if err != nil {
				return err
			}
		}
	default:
		fmt.Println("objType.Kind:", objType.Kind(), "objValue.kind:", objValue.Kind())
		fmt.Printf("%#v\n", obj)
	}

	return nil
}

func setCty(body *hclwrite.Body, key string, c cty.Value) {
	if c.Type() == cty.String {
		s := c.AsString()
		if strings.ContainsAny(s, "\n") {
			if !strings.HasSuffix(s, "\n") {
				s = s + "\n"
			}

			// we do non-indentent heredocs because figuring out the indentation
			// level requires passing it everywhere.
			body.SetAttributeRaw(key, hclwrite.Tokens{
				{Type: hclsyntax.TokenOHeredoc, Bytes: []byte(`<<HEREDOC`), SpacesBefore: 0},
				{Type: hclsyntax.TokenStringLit, Bytes: []byte("\n" + s), SpacesBefore: 0},
				{Type: hclsyntax.TokenCHeredoc, Bytes: []byte(`HEREDOC`), SpacesBefore: 0},
			})
		} else {
			s = strings.ReplaceAll(s, `"`, `\"`)
			body.SetAttributeRaw(key, hclwrite.Tokens{
				{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`), SpacesBefore: 0},
				{Type: hclsyntax.TokenStringLit, Bytes: []byte(s), SpacesBefore: 0},
				{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`), SpacesBefore: 0},
			})
		}
	} else {
		body.SetAttributeValue(key, c)
	}
}

func convertValue(value interface{}) (cv cty.Value, ok bool) {
	switch v := value.(type) {
	case string:
		if v != "" {
			return cty.StringVal(v), true
		}
	case *string:
		if v != nil {
			return cty.StringVal(*v), true
		}
	case int8:
		return cty.NumberIntVal(int64(v)), true
	case int:
		return cty.NumberIntVal(int64(v)), true
	case *int:
		if v != nil {
			return cty.NumberIntVal(int64(*v)), true
		}
	case *bool:
		if v != nil {
			return cty.BoolVal(*v), true
		}
	case nil:
		return cty.NilVal, true
	case bool:
		return cty.BoolVal(v), true
	case time.Duration:
		return cty.StringVal(v.String()), true
	case *time.Duration:
		if v != nil {
			return cty.StringVal(v.String()), true
		}
	case []interface{}:
		if len(v) == 0 {
			return cty.ListValEmpty(cty.String), true
		} else {
			list := []cty.Value{}
			for _, value := range v {
				converted, ok := convertValue(value)
				if !ok {
					fmt.Printf("Couldn't convert value: %T\n", value)
					return cv, false
				}
				list = append(list, converted)
			}
			return cty.ListVal(list), true
		}
	case []string:
		if len(v) != 0 {
			converted := []cty.Value{}
			for _, value := range v {
				converted = append(converted, cty.StringVal(value))
			}
			return cty.ListVal(converted), true
		}
	case map[string]string:
		if len(v) != 0 {
			converted := map[string]cty.Value{}
			for mk, mv := range v {
				converted[mk] = cty.StringVal(mv)
			}
			return cty.MapVal(converted), true
		}
	case map[string][]string:
		if len(v) != 0 {
			convertedMap := map[string]cty.Value{}
			for mk, mv := range v {
				convertedList, ok := convertValue(mv)
				if !ok {
					return cv, false
				}
				convertedMap[mk] = convertedList
			}
			return cty.MapVal(convertedMap), true
		}
	case map[string]interface{}:
		if len(v) != 0 {
			convertedMap := map[string]cty.Value{}
			for mk, mv := range v {
				converted, ok := convertValue(mv)
				if !ok {
					return cv, false
				}
				convertedMap[mk] = converted
			}
			return cty.ObjectVal(convertedMap), true
		}
	default:
		panic(fmt.Sprintf("Unknown type for: %t, %#t %#v", v, v, v))
	}

	return
}
