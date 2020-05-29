package runtime

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
)

type NodeSpec struct {
	ID         string     `json:"id"`
	Icon       string     `json:"icon"`
	Name       string     `json:"name"`
	Color      string     `json:"color"`
	Editor     *string    `json:"editor"`
	Inputs     int        `json:"inputs"`
	Outputs    int        `json:"outputs"`
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
	Type         string                  `json:"type"`
	Title        string                  `json:"title"`
	SubTitle     string                  `json:"subtitle"`
	Category     *int                    `json:"category,omitempty"`
	Properties   *map[string]interface{} `json:"properties,omitempty"`
	CsScope      *bool                   `json:"csScope,omitempty"`
	JsScope      *bool                   `json:"jsScope,omitempty"`
	CustomScope  *bool                   `json:"customScope,omitempty"`
	MessageScope *bool                   `json:"messageScope,omitempty"`
	MessageOnly  *bool                   `json:"messageOnly,omitempty"`
	Multiple     *bool                   `json:"multiple,omitempty"`
	VariableType string                  `json:"variableType,omitempty"`
	Enum         []interface{}           `json:"enum,omitempty"`
	EnumNames    []string                `json:"enumNames,omitempty"`
}

type VarDataProperty struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

func generateSpecFile(pluginName, version string) {

	var nodes []NodeSpec
	types := GetNodeTypes()

	for _, t := range types {
		snode, _ := t.FieldByName("SNode")
		id := snode.Tag.Get("id")
		name := snode.Tag.Get("name")
		icon := snode.Tag.Get("icon")
		color := snode.Tag.Get("color")
		editor := snode.Tag.Get("editor")
		inputs, hasInputs := snode.Tag.Lookup("inputs")
		outputs, hasOutputs := snode.Tag.Lookup("outputs")

		if !hasInputs {
			inputs = "1"
		}
		if !hasOutputs {
			outputs = "1"
		}

		spec := NodeSpec{ID: id, Name: name, Icon: icon, Color: color}
		spec.Inputs, _ = strconv.Atoi(inputs)
		spec.Outputs, _ = strconv.Atoi(outputs)
		if editor != "" {
			spec.Editor = &editor
		}

		inProperty := Property{FormData: make(map[string]interface{}), UISchema: make(map[string]interface{})}
		outProperty := Property{FormData: make(map[string]interface{}), UISchema: make(map[string]interface{})}
		optProperty := Property{FormData: make(map[string]interface{}), UISchema: make(map[string]interface{})}

		inProperty.UISchema["ui:order"] = []string{}
		outProperty.UISchema["ui:order"] = []string{}
		optProperty.UISchema["ui:order"] = []string{}

		inProperty.Schema = Schema{Title: "Input", Type: "object", Properties: make(map[string]SProperty)}
		outProperty.Schema = Schema{Title: "Output", Type: "object", Properties: make(map[string]SProperty)}
		optProperty.Schema = Schema{Title: "Option", Type: "object", Properties: make(map[string]SProperty)}

		for i := 0; i < t.NumField(); i++ {

			field := t.Field(i)
			fieldName := field.Name
			title := field.Tag.Get("title")
			enum := field.Tag.Get("enum")

			sProp := SProperty{Title: title}
			isVar := field.Type == reflect.TypeOf(Variable{})
			isCred := field.Type == reflect.TypeOf(Credentials{})
			isEnum := len(enum) > 0

			if isVar {
				sProp.Type = "object"
				sProp.VariableType = getVariableType(field)
				sProp.Properties = &map[string]interface{}{"scope": map[string]string{"type": "string"}, "name": map[string]string{"type": "string"}}

			} else if isCred {
				category, _ := strconv.Atoi(field.Tag.Get("category"))
				sProp.Type = "object"
				sProp.SubTitle = "Credentials"
				sProp.Category = &category
				sProp.Properties = &map[string]interface{}{"vaultId": map[string]string{"type": "string"}, "itemId": map[string]string{"type": "string"}}

			} else if isEnum {
				sProp.Type = field.Tag.Get("type")
				json.Unmarshal([]byte(enum), &sProp.Enum)
				json.Unmarshal([]byte(field.Tag.Get("enumNames")), &sProp.EnumNames)
				multiple := true
				sProp.Multiple = &multiple

			} else {
				sProp.Type = strings.ToLower(getVariableType(field))
			}

			_, csScope := field.Tag.Lookup("csScope")
			_, customScope := field.Tag.Lookup("customScope")
			_, jsScope := field.Tag.Lookup("jsScope")
			_, messageScope := field.Tag.Lookup("messageScope")
			_, messageOnly := field.Tag.Lookup("messageOnly")
			_, isHidden := field.Tag.Lookup("hidden")

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
			if messageOnly {
				sProp.MessageOnly = &messageOnly
			}

			lowerFieldName := lowerFirstLetter(fieldName)
			if strings.HasPrefix(fieldName, "In") { // input

				inProperty.Schema.Properties[lowerFieldName] = sProp
				inProperty.UISchema["ui:order"] = append(inProperty.UISchema["ui:order"].([]string), lowerFieldName)

				if isHidden {
					inProperty.UISchema[lowerFieldName] = map[string]string{"ui:widget": "hidden"}
				}

				if isVar {
					inProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					inProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				} else if isCred {
					inProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "credentials"}
				} else {
					inProperty.FormData[lowerFieldName] = field.Tag.Get("name")
				}

			} else if strings.HasPrefix(fieldName, "Out") { // output

				outProperty.Schema.Properties[lowerFieldName] = sProp
				outProperty.UISchema["ui:order"] = append(outProperty.UISchema["ui:order"].([]string), lowerFieldName)

				if isHidden {
					outProperty.UISchema[lowerFieldName] = map[string]string{"ui:widget": "hidden"}
				}

				if isVar {
					outProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					outProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				} else if isCred {
					outProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "credentials"}
				} else {
					outProperty.FormData[lowerFieldName] = field.Tag.Get("name")
				}

			} else if strings.HasPrefix(fieldName, "Opt") { // option

				optProperty.Schema.Properties[lowerFieldName] = sProp
				optProperty.UISchema["ui:order"] = append(optProperty.UISchema["ui:order"].([]string), lowerFieldName)

				if isHidden {
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:widget": "hidden"}
				}

				if isVar {
					optProperty.FormData[lowerFieldName] = VarDataProperty{Scope: field.Tag.Get("scope"), Name: field.Tag.Get("name")}
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				} else if isCred {
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "credentials"}
				} else {
					optProperty.FormData[lowerFieldName] = field.Tag.Get("name")
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
