package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"html/template"
	"strings"
	"syscall/js"
	"unicode"
)

func main() {
	c := make(chan struct{})
	registerFuncs()
	<-c
}

func registerFuncs() {
	js.Global().Set("generateCode", js.FuncOf(generator))
	js.Global().Set("helloWorld", js.FuncOf(helloWorld))
}

func helloWorld(this js.Value, params []js.Value) interface{} {
	return "Hello World"
}

type Final struct {
	PackageName    string
	StructsDetails []StructDef
}

type StructDef struct {
	Name   string
	Fields []Field
}

type Field struct {
	Name string
	Tag  string
	Type string
}

const (
	textTemplate = `package {{.PackageName}}
	{{range $struct := .StructsDetails }}type {{ $struct.Name }} struct {
		{{ range $field := .Fields }}{{ $field.Name }} {{ $field.Type }} ` + "{{if ne $field.Tag \"\" }}`json:\"{{$field.Tag }}\"`{{end}}" + `
		{{end}}
	}

	func (s *{{ $struct.Name }}) String() string {
		return "{{ range $field := .Fields }}{{ $field.Name }}: s.{{ $field.Name }}{{ end }}"
	}
	{{end}}`
)

var buf bytes.Buffer

func generator(this js.Value, params []js.Value) interface{} {
	data := make([]byte, params[0].Get("length").Int())

	js.CopyBytesToGo(data, params[0])

	buf = *bytes.NewBuffer(data)

	details, err := prepareData()
	if err != nil {
		return err.Error()
	}

	res, err := ExecuteTemplate(details)
	if err != nil {
		return err.Error()
	}

	fmt.Println(res.String())

	fmtRes, err := format.Source(res.Bytes())
	if err != nil {
		return err.Error()
	}

	return string(fmtRes)
}

func prepareData() (structsDetails Final, err error) {
	fSet := token.NewFileSet()
	structsDetails.StructsDetails = make([]StructDef, 0)

	f, err := parser.ParseFile(fSet, "", buf.Bytes(), parser.AllErrors)
	if err != nil {
		return
	}

	structsDetails.PackageName = f.Name.Name

	ast.Inspect(f, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.TypeSpec:
			e, ok := t.Type.(*ast.StructType)
			if !ok {
				return false
			}
			sd := StructDef{
				Name: t.Name.Name,
			}

			fields := make([]Field, 0)

			for _, field := range e.Fields.List {
				fields = append(fields, Field{
					Name: field.Names[0].Name,
					Type: string(buf.Bytes()[field.Type.Pos()-1 : field.Type.End()-1]),
					Tag:  GenerateTagName(field.Names[0].Name),
				})
			}
			sd.Fields = fields
			structsDetails.StructsDetails = append(structsDetails.StructsDetails, sd)
		}
		return true
	})

	return

}

func GenerateTagName(fieldName string) (res string) {

	tag := strings.Builder{}
	lastCapLetterIdx := 0
	for i, c := range fieldName {
		// if the field is exported only then generate json tag
		if unicode.IsUpper(rune(fieldName[0])) {
			if unicode.IsUpper(c) {
				lastCapLetterIdx = i
				if lastCapLetterIdx != 0 {
					_, err := tag.WriteRune('_')
					if err != nil {
						return fieldName
					}
				}
				_, err := tag.WriteRune(unicode.ToLower(c))
				if err != nil {
					return fieldName
				}
			} else {
				_, err := tag.WriteRune(c)
				if err != nil {
					return fieldName
				}
			}
		}
	}
	res = tag.String()
	return res
}

func ExecuteTemplate(sds Final) (resp bytes.Buffer, err error) {
	t := template.New("tmpl")
	t, err = t.Parse(textTemplate)
	if err != nil {
		return bytes.Buffer{}, err
	}

	err = t.Execute(&resp, sds)
	if err != nil {
		fmt.Println(err.Error())
		return bytes.Buffer{}, err
	}

	return resp, err
}
