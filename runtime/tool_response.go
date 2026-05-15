package runtime

import (
	"github.com/robomotionio/robomotion-go/message"
)

// IsToolRequest checks if the current message is a tool request
func IsToolRequest(ctx message.Context) bool {
	msgType := ctx.Get("__message_type__")
	return msgType == "tool_request"
}

// ToolName returns the name of the tool the agent invoked for this
// message. For single-Tool nodes the name matches the node's tool tag and
// can be ignored. For Toolkit nodes this is the discriminator used to
// dispatch inside OnMessage. Empty string when ctx is not a tool request.
func ToolName(ctx message.Context) string {
	if v, ok := ctx.Get("__tool_name__").(string); ok {
		return v
	}
	return ""
}

// ToolParameters returns the parsed parameters object the agent passed
// with the tool call, or an empty map. Empty when ctx is not a tool
// request. Use the returned map directly, or json.Marshal+Unmarshal it
// into a typed struct if the schema is stable.
func ToolParameters(ctx message.Context) map[string]interface{} {
	v := ctx.Get("__parameters__")
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// ToolResponse sends a response back to the LLM Agent and prevents message flow
func ToolResponse(ctx message.Context, status string, data map[string]interface{}, errorMsg string) error {
	if !IsToolRequest(ctx) {
		return nil // Not a tool request
	}

	callerID := ctx.Get("__tool_caller_id__")
	agentNodeID := ctx.Get("__agent_node_id__")
	
	// Create response context with required fields
	responseCtx := message.NewContext([]byte("{}"))
	
	// Copy all essential fields from the original message
	// This ensures the LLM Agent has all necessary context to continue
	if msgID := ctx.Get("id"); msgID != nil {
		responseCtx.Set("id", msgID)
	}
	
	// Copy session information if present
	if sessionID := ctx.Get("session_id"); sessionID != nil {
		responseCtx.Set("session_id", sessionID)
	}
	
	// Copy any query information
	if query := ctx.Get("query"); query != nil {
		responseCtx.Set("query", query)
	}
	
	// Set tool response specific fields
	responseCtx.Set("__message_type__", "tool_response")
	responseCtx.Set("__tool_caller_id__", callerID)
	responseCtx.Set("__tool_status__", status)
	
	if errorMsg != "" {
		responseCtx.Set("__tool_error__", errorMsg)
	}
	if data != nil {
		responseCtx.Set("__tool_data__", data)
	}
	
	// Send response back to LLM Agent
	if agentID, ok := agentNodeID.(string); ok && agentID != "" {
		data, _ := responseCtx.GetRaw()
		EmitInput(agentID, data)
	}
	
	// Prevent message flow to next node by clearing context
	ctx.SetRaw(nil)
	
	return nil
}