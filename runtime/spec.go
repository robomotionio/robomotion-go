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
	SubTitle     *string                 `json:"subtitle,omitempty"`
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
		Node, _ := t.FieldByName("Node")
		nodeSpec := Node.Tag.Get("spec")
		nsMap := parseSpec(nodeSpec)

		id := nsMap["id"]
		name := nsMap["name"]
		icon := Icons[nsMap["icon"]]
		color := nsMap["color"]
		editor := nsMap["editor"]
		inputs, hasInputs := nsMap["inputs"]
		outputs, hasOutputs := nsMap["outputs"]

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
		optProperty.Schema = Schema{Title: "Options", Type: "object", Properties: make(map[string]SProperty)}

		for i := 0; i < t.NumField(); i++ {

			field := t.Field(i)
			fieldName := field.Name

			fieldSpec := field.Tag.Get("spec")
			fsMap := parseSpec(fieldSpec)

			title, hasTitle := fsMap["title"]
			enum := fsMap["enum"]

			if !hasTitle {
				title = fieldName
			}

			sProp := SProperty{Title: title}
			isVar := field.Type == reflect.TypeOf(InVariable{}) || field.Type == reflect.TypeOf(OutVariable{}) || field.Type == reflect.TypeOf(OptVariable{})
			isCred := field.Type == reflect.TypeOf(Credential{})
			isEnum := len(enum) > 0

			if isVar {
				sProp.Type = "object"
				sProp.VariableType = getVariableType(field, fsMap)
				sProp.Properties = &map[string]interface{}{"scope": map[string]string{"type": "string"}, "name": map[string]string{"type": "string"}}

			} else if isCred {
				category, _ := strconv.Atoi(fsMap["category"])
				sProp.Type = "object"
				subtitle := "Credentials"
				sProp.SubTitle = &subtitle
				sProp.Category = &category
				sProp.Properties = &map[string]interface{}{"vaultId": map[string]string{"type": "string"}, "itemId": map[string]string{"type": "string"}}

			} else if isEnum {
				sProp.Enum, sProp.EnumNames = parseEnum(enum, fsMap["enumNames"], getVariableType(field, fsMap))
				sProp.Type = strings.ToLower(getVariableType(field, fsMap))
				multiple := true
				sProp.Multiple = &multiple

			} else {
				sProp.Type = strings.ToLower(getVariableType(field, fsMap))
			}

			_, csScope := fsMap["csScope"]
			_, customScope := fsMap["customScope"]
			_, jsScope := fsMap["jsScope"]
			_, messageScope := fsMap["messageScope"]
			_, messageOnly := fsMap["messageOnly"]
			_, isHidden := fsMap["hidden"]

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

			_, isInput := fsMap["input"]
			_, isOutput := fsMap["output"]
			_, isOption := fsMap["option"]

			lowerFieldName := lowerFirstLetter(fieldName)

			scope, hasScope := fsMap["scope"]
			n, hasName := fsMap["name"]
			if isVar && scope == "Message" && !hasName {
				n = lowerFieldName
			}

			if isVar && !hasScope {
				scope = "Custom"
				n = ""
			}

			if field.Type == reflect.TypeOf(InVariable{}) || isInput { // input

				inProperty.Schema.Properties[lowerFieldName] = sProp
				inProperty.UISchema["ui:order"] = append(inProperty.UISchema["ui:order"].([]string), lowerFieldName)

				if isHidden {
					inProperty.UISchema[lowerFieldName] = map[string]string{"ui:widget": "hidden"}
				}

				if isVar {
					inProperty.FormData[lowerFieldName] = VarDataProperty{Scope: scope, Name: n}
					inProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				} else {
					inProperty.FormData[lowerFieldName] = n
				}

			} else if field.Type == reflect.TypeOf(OutVariable{}) || isOutput { // output

				outProperty.Schema.Properties[lowerFieldName] = sProp
				outProperty.UISchema["ui:order"] = append(outProperty.UISchema["ui:order"].([]string), lowerFieldName)

				if isHidden {
					outProperty.UISchema[lowerFieldName] = map[string]string{"ui:widget": "hidden"}
				}

				if isVar {
					outProperty.FormData[lowerFieldName] = VarDataProperty{Scope: scope, Name: n}
					outProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				} else {
					outProperty.FormData[lowerFieldName] = n
				}

			} else if field.Type == reflect.TypeOf(OptVariable{}) || isCred || isEnum { // option

				optProperty.Schema.Properties[lowerFieldName] = sProp
				optProperty.UISchema["ui:order"] = append(optProperty.UISchema["ui:order"].([]string), lowerFieldName)

				if isHidden {
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:widget": "hidden"}
				}

				if isVar {
					optProperty.FormData[lowerFieldName] = VarDataProperty{Scope: scope, Name: n}
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "variable"}
				} else if isCred {
					optProperty.UISchema[lowerFieldName] = map[string]string{"ui:field": "credentials"}
				} else if isEnum {
					v := fsMap["value"]
					optProperty.FormData[lowerFieldName] = parseValue(field, v)
				} else {
					optProperty.FormData[lowerFieldName] = n
				}

			} else if isOption {

				optProperty.Schema.Properties[lowerFieldName] = sProp
				optProperty.UISchema["ui:order"] = append(optProperty.UISchema["ui:order"].([]string), lowerFieldName)
				v := fsMap["value"]
				optProperty.FormData[lowerFieldName] = parseValue(field, v)

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

func parseEnum(enum, enumNames, enumType string) ([]interface{}, []string) {
	var (
		enumArr     []interface{}
		enumNameArr []string
	)

	enumParts := strings.Split(enum, "|")
	if enumType == "Integer" {
		for _, part := range enumParts {
			d, err := strconv.Atoi(part)
			if err != nil {
				d = 0
			}
			enumArr = append(enumArr, d)
		}

	} else if enumType == "Double" {
		for _, part := range enumParts {
			d, err := strconv.ParseFloat(part, 64)
			if err != nil {
				d = 0
			}
			enumArr = append(enumArr, d)
		}

	} else {
		for _, part := range enumParts {
			enumArr = append(enumArr, part)
		}
	}

	enumNameArr = strings.Split(enumNames, "|")
	return enumArr, enumNameArr
}

func parseSpec(spec string) map[string]string {
	nsMap := map[string]string{}

	kvs := strings.Split(spec, ",")
	for _, kv := range kvs {
		p := strings.Split(kv, "=")
		if len(p) < 2 {
			nsMap[p[0]] = ""
			continue
		}

		k, v := p[0], p[1]
		nsMap[k] = v
	}

	return nsMap
}

func parseValue(f reflect.StructField, v string) interface{} {

	var cv interface{}
	switch f.Type.Kind() {
	case reflect.Bool:
		cv, _ = strconv.ParseBool(v)
	case reflect.Int, reflect.Int64:
		cv, _ = strconv.ParseInt(v, 10, 64)
	case reflect.Int32:
		cv, _ = strconv.ParseInt(v, 10, 32)
	case reflect.Float32:
		cv, _ = strconv.ParseFloat(v, 32)
	case reflect.Float64:
		cv, _ = strconv.ParseFloat(v, 64)
	case reflect.String:
		cv = v
	}

	return cv
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

func getVariableType(f reflect.StructField, fsMap map[string]string) string {

	if f.Type == reflect.TypeOf(Variable{}) {
		return upperFirstLetter(strings.ToLower(fsMap["type"]))
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
