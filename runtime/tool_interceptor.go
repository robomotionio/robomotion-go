package runtime

import (
	"reflect"
	"strings"

	"github.com/robomotionio/robomotion-go/message"
)

// ToolInterceptor wraps a MessageHandler to automatically handle tool requests
type ToolInterceptor struct {
	originalHandler MessageHandler
	nodeType        reflect.Type
	hasTool         bool
}

// NewToolInterceptor creates a tool interceptor for a node
func NewToolInterceptor(handler MessageHandler) MessageHandler {
	nodeType := reflect.TypeOf(handler)
	if nodeType.Kind() == reflect.Ptr {
		nodeType = nodeType.Elem()
	}

	// Check if this node has runtime.Tool
	hasTool := false
	if _, hasToolField := nodeType.FieldByName("Tool"); hasToolField {
		hasTool = true
	}

	return &ToolInterceptor{
		originalHandler: handler,
		nodeType:        nodeType,
		hasTool:         hasTool,
	}
}

func (ti *ToolInterceptor) OnCreate() error {
	return ti.originalHandler.OnCreate()
}

func (ti *ToolInterceptor) OnMessage(ctx message.Context) error {
	// Check if this is a tool request and this node supports tools
	if ti.hasTool && IsToolRequest(ctx) {
		return ti.handleToolRequest(ctx)
	}

	// Pass through to original handler for normal processing
	return ti.originalHandler.OnMessage(ctx)
}

func (ti *ToolInterceptor) OnClose() error {
	return ti.originalHandler.OnClose()
}

// handleToolRequest automatically processes tool requests
func (ti *ToolInterceptor) handleToolRequest(ctx message.Context) error {
	// Call the original handler to do the actual work
	err := ti.originalHandler.OnMessage(ctx)
	
	// If the original handler didn't call ToolResponse, send a default response
	if !hasToolResponseBeenSent(ctx) {
		if err != nil {
			return ToolResponse(ctx, "error", nil, err.Error())
		} else {
			// Collect output variables automatically
			outputData := ti.collectOutputVariables(ctx)
			return ToolResponse(ctx, "success", outputData, "")
		}
	}
	
	return err
}

// collectOutputVariables automatically collects output from the node
func (ti *ToolInterceptor) collectOutputVariables(ctx message.Context) map[string]interface{} {
	outputData := make(map[string]interface{})
	
	// Use reflection to find OutVariable fields and collect their values
	nodeValue := reflect.ValueOf(ti.originalHandler)
	if nodeValue.Kind() == reflect.Ptr {
		nodeValue = nodeValue.Elem()
	}
	
	nodeType := nodeValue.Type()
	for i := 0; i < nodeType.NumField(); i++ {
		field := nodeType.Field(i)
		
		// Check if this is an OutVariable field
		if field.Type.Kind() == reflect.Struct {
			typeName := field.Type.String()
			if strings.HasSuffix(typeName, "OutVariable") {
				// Extract the variable name from spec tag
				specTag := field.Tag.Get("spec")
				if name := extractNameFromSpec(specTag); name != "" {
					// Try to get the value from context (it should have been set)
					if value := ctx.Get(name); value != nil {
						outputData[name] = value
					}
				}
			}
		}
	}
	
	// If no output variables found, return basic status
	if len(outputData) == 0 {
		outputData["status"] = "completed"
	}
	
	return outputData
}

// extractNameFromSpec extracts the name parameter from spec tag
func extractNameFromSpec(spec string) string {
	// Use existing parseSpec function
	parts := parseSpec(spec)
	return parts["name"]
}

// hasToolResponseBeenSent checks if ToolResponse was already called
func hasToolResponseBeenSent(ctx message.Context) bool {
	// If context is nil, ToolResponse was called (it sets ctx.SetRaw(nil))
	raw, err := ctx.GetRaw()
	return err != nil || raw == nil
}