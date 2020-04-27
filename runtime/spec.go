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
	Type         string     `json:"type"`
	Title        string     `json:"title"`
	Properties   *VProperty `json:"properties,omitempty"`
	CsScope      *bool      `json:"csScope,omitempty"`
	JsScope      *bool      `json:"jsScope,omitempty"`
	CustomScope  *bool      `json:"customScope,omitempty"`
	MessageScope *bool      `json:"messageScope,omitempty"`
	VariableType string     `json:"variableType,omitempty"`
}

type VProperty struct {
	Name  *Type `json:"name,omitempty"`
	Scope *Type `json:"scope,omitempty"`
}

type VarDataProperty struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type Type struct {
	Type string `json:"type,omitempty"`
}

func generateSpecFile(pluginName, version string) {

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
				sProp.VariableType = getVariableType(field)
				sProp.Properties = &VProperty{Name: &Type{Type: "string"}, Scope: &Type{Type: "string"}}

			} else {
				sProp.Type = strings.ToLower(getVariableType(field))
			}

			_, csScope := field.Tag.Lookup("csScope")
			_, customScope := field.Tag.Lookup("customScope")
			_, jsScope := field.Tag.Lookup("jsScope")
			_, messageScope := field.Tag.Lookup("messageScope")

			if csScope {
				sProp.CsScope = &csScope
			}
			if customScope {
				sProp.CustomScope = &customScope
			}
			if jsScope {
				sProp.JsScope = &jsScope
			}
			if messageScope {
				sProp.MessageScope = &messageScope
			}

			lowerFieldName := lowerFirstLetter(fieldName)
			if strings.HasPrefix(fieldName, "In") { // input

				inProperty.Schema.Properties[lowerFieldName] = sProp
				inProperty.UISchema["ui:order"] = []string{"*"} // FIX
				if isVar {
					inProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					inProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				}

			} else if strings.HasPrefix(fieldName, "Out") { // output

				outProperty.Schema.Properties[lowerFieldName] = sProp
				outProperty.UISchema["ui:order"] = []string{"*"} // FIX

				if isVar {
					outProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					outProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				}

			} else if strings.HasPrefix(fieldName, "Opt") { // option

				optProperty.Schema.Properties[lowerFieldName] = sProp
				optProperty.UISchema["ui:order"] = []string{"*"} // FIX
				if isVar {
					optProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
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

	data := map[string]interface{}{"nodes": nodes, "name": pluginName, "version": version}
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

func upperFirstLetter(text string) string {
	if len(text) < 2 {
		return strings.ToUpper(text)
	}

	return fmt.Sprintf("%s%s", strings.ToUpper(text[:1]), text[1:])
}

func getVariableType(f reflect.StructField) string {

	if f.Type == reflect.TypeOf(Variable{}) {
		return upperFirstLetter(strings.ToLower(f.Tag.Get("type")))
	}

	kinds := map[reflect.Kind]string{
		reflect.Bool: "Boolean", reflect.Int: "Integer", reflect.Int8: "Integer", reflect.Int16: "Integer", reflect.Int32: "Integer", reflect.Int64: "Integer",
		reflect.Uint: "Integer", reflect.Uint8: "Integer", reflect.Uint16: "Integer", reflect.Uint32: "Integer", reflect.Uint64: "Integer", reflect.Float32: "Double",
		reflect.Float64: "Double", reflect.Array: "Array",
	}

	if s, ok := kinds[f.Type.Kind()]; ok {
		return s
	}

	return "String"
}
