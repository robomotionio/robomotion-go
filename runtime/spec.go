package runtime

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
)

type NodeSpec struct {
	ID         string     `json:"id"`
	Icon       string     `json:"icon"`
	Name       string     `json:"name"`
	Color      string     `json:"color"`
	Inputs     byte       `json:"inputs"`
	Outputs    byte       `json:"outputs"`
	Properties []Property `json:"properties"`
}

type Property struct {
	Schema   Schema                 `json:"schema"`
	FormData map[string]interface{} `json:"formData"`
	UISchema map[string]interface{} `json:"uiSchema"`
}

type Schema struct {
	Type       string               `json:"type"`
	Title      string               `json:"title"`
	Properties map[string]SProperty `json:"properties"`
}

type SProperty struct {
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Properties   VProperty `json:"properties,omitempty"`
	CsScope      bool      `json:"csScope"`
	JsScope      bool      `json:"jsScope"`
	CustomScope  bool      `json:"customScope"`
	MessageScope bool      `json:"messageScope"`
	VariableType string    `json:"variableType,omitempty"`
}

type VProperty struct {
	Name  Type `json:"name"`
	Scope Type `json:"scope"`
}

type VarDataProperty struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type Type struct {
	Type string `json:"type"`
}

func generateSpecFile() {

	var nodes []NodeSpec
	types := GetNodeTypes()
	for _, t := range types {
		snode, _ := t.FieldByName("SNode")
		id := snode.Tag.Get("id")
		name := snode.Tag.Get("name")

		spec := NodeSpec{ID: id, Name: name, Inputs: 1, Outputs: 1} // FIX

		inProperty := Property{FormData: make(map[string]interface{}), UISchema: make(map[string]interface{})}
		outProperty := Property{FormData: make(map[string]interface{}), UISchema: make(map[string]interface{})}
		optProperty := Property{FormData: make(map[string]interface{}), UISchema: make(map[string]interface{})}

		inProperty.Schema = Schema{Title: "Input", Type: "object", Properties: make(map[string]SProperty)}
		outProperty.Schema = Schema{Title: "Output", Type: "object", Properties: make(map[string]SProperty)}
		optProperty.Schema = Schema{Title: "Option", Type: "object", Properties: make(map[string]SProperty)}

		for i := 0; i < t.NumField(); i++ {

			field := t.Field(i)
			fieldName := field.Name
			title := field.Tag.Get("title")

			sProp := SProperty{Title: title}
			isVar := field.Type == reflect.TypeOf(Variable{})
			if isVar {
				sProp.Type = "object"
				sProp.VariableType = getVariableType(field.Type)
				sProp.Properties = VProperty{Name: Type{Type: "string"}, Scope: Type{Type: "string"}}

			} else {
				sProp.Type = strings.ToLower(getVariableType(field.Type))
			}

			_, sProp.CsScope = field.Tag.Lookup("csScope")
			_, sProp.CustomScope = field.Tag.Lookup("customScope")
			_, sProp.JsScope = field.Tag.Lookup("jsScope")
			_, sProp.MessageScope = field.Tag.Lookup("messageScope")

			lowerFieldName := lowerFirstLetter(fieldName)
			if strings.HasPrefix(fieldName, "In") { // input

				inProperty.Schema.Properties[lowerFieldName] = sProp
				if isVar {
					inProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					inProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
					inProperty.UISchema["ui:order"] = []string{"*"} // FIX
				}

			} else if strings.HasPrefix(fieldName, "Out") { // output

				outProperty.Schema.Properties[lowerFieldName] = sProp
				if isVar {
					outProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					outProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
					outProperty.UISchema["ui:order"] = []string{"*"} // FIX
				}

			} else if strings.HasPrefix(fieldName, "Opt") { // option

				optProperty.Schema.Properties[lowerFieldName] = sProp
				if isVar {
					optProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
					optProperty.UISchema["ui:order"] = []string{"*"} // FIX
				}
			}
		}

		if len(inProperty.Schema.Properties) > 0 {
			spec.Properties = append(spec.Properties, inProperty)
		}

		if len(outProperty.Schema.Properties) > 0 {
			spec.Properties = append(spec.Properties, outProperty)
		}

		if len(optProperty.Schema.Properties) > 0 {
			spec.Properties = append(spec.Properties, optProperty)
		}

		nodes = append(nodes, spec)
	}

	data := map[string]interface{}{"nodes": nodes}
	d, err := json.Marshal(data)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(d))
}

func lowerFirstLetter(text string) string {
	if len(text) < 2 {
		return strings.ToLower(text)
	}

	return fmt.Sprintf("%s%s", strings.ToLower(text[:1]), text[1:])
}

func getVariableType(t reflect.Type) string {

	kinds := map[reflect.Kind]string{
		reflect.Bool: "Boolean", reflect.Int: "Integer", reflect.Int8: "Integer", reflect.Int16: "Integer", reflect.Int32: "Integer", reflect.Int64: "Integer",
		reflect.Uint: "Integer", reflect.Uint8: "Integer", reflect.Uint16: "Integer", reflect.Uint32: "Integer", reflect.Uint64: "Integer", reflect.Float32: "Double",
		reflect.Float64: "Double", reflect.Array: "Array",
	}

	if s, ok := kinds[t.Kind()]; ok {
		return s
	}

	return "String"
}
