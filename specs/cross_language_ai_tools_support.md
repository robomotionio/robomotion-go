# Technical Deep Dive: Cross-Language AI Tools Flow

## Overview: Go Send Mail Node ↔ Python LLM Agent

This document provides a complete technical walkthrough of how a Go-based Send Mail node becomes an AI tool for a Python LLM Agent through the Robomotion cross-language AI tools system.

## Step 1: Node Definition (Go Side)

```go
type SendMailNode struct {
    runtime.Node `spec:"id=Package.SendMail,name=Send Mail"`
    runtime.Tool `tool:"name=send_email,description=Send email messages"`
    
    // Parameters with aiScope option - Designer can choose to let AI control these
    InTo      runtime.InVariable[string] `spec:"scope=AI,name=to,aiScope,customScope,messageScope,description=Email recipient"`
    InBody    runtime.InVariable[string] `spec:"scope=AI,name=body,aiScope,customScope,messageScope,description=Email body"`
    InSubject runtime.InVariable[string] `spec:"scope=AI,name=subject,aiScope,customScope,messageScope,description=Email subject"`
    
    // Parameters without aiScope option - Designer can NEVER let AI control these  
    InSMTPServer runtime.InVariable[string] `spec:"scope=Custom,name=smtp.company.com,customScope,messageScope"`
}

// ZERO tool-specific code needed! Just normal node logic
func (n *SendMailNode) OnMessage(ctx message.Context) error {
    // Just normal processing - runtime automatically handles tool vs RPA context
    to, _ := n.InTo.Get(ctx)      // Gets from LLM OR previous node automatically
    body, _ := n.InBody.Get(ctx)  // Gets from LLM OR previous node automatically  
    subject, _ := n.InSubject.Get(ctx) // Gets from LLM OR Designer config automatically
    
    // Never AI-controllable (no aiScope option)
    server, _ := n.InSMTPServer.Get(ctx)  // Always from Designer config
    
    // Normal email sending logic - same for both RPA and tool usage
    messageID, err := sendEmail(server, from, to, subject, body)
    if err != nil {
        return err // Runtime automatically converts to tool error if needed
    }
    
    // Set outputs - runtime automatically collects for tool responses
    n.OutSuccess.Set(ctx, true)
    n.OutMessageID.Set(ctx, messageID)
    
    return nil // Runtime handles everything automatically
}
```

## Step 2: Spec Generation & Registration (Go Runtime)

When the Go package starts:

```go
// runtime/spec.go processes the struct tags
func GenerateSpec(nodeType reflect.Type) NodeSpec {
    spec := NodeSpec{}
    
    // Parse the runtime.Tool tag
    if toolField, hasToolField := nodeType.FieldByName("Tool"); hasToolField {
        toolTag := toolField.Tag.Get("tool") // "name=send_email,description=Send email"
        spec.Tool = map[string]string{
            "name": "send_email",
            "description": "Send email messages"
        }
    }
    
    // Parse all InVariable fields and their scopes
    for each field with InVariable[T] {
        fieldSpec := parseFieldTag(field.Tag) // Extract scope=AI, name=to, etc.
        spec.Properties = append(spec.Properties, fieldSpec)
    }
    
    return spec
}
```

**Generated Spec (JSON)**:
```json
{
  "id": "Package.SendMail",
  "tool": {
    "name": "send_email", 
    "description": "Send email messages"
  },
  "properties": [
    {"name": "to", "type": "string", "scope": "AI", "aiScope": true, "description": "Email recipient"},
    {"name": "body", "type": "string", "scope": "AI", "aiScope": true, "description": "Email body"},
    {"name": "subject", "type": "string", "scope": "AI", "aiScope": true, "description": "Email subject"},
    {"name": "smtp_server", "type": "string", "scope": "Custom", "aiScope": false, "value": "smtp.company.com"}
  ]
}
```

## Step 3: Tool Discovery (Python Agent)

When Python LLM Agent initializes:

```python
# agents/nodes/simulation/tool_simulation.py
def connect_tools(parent_agent_node_id: str, tools_port_index: int) -> List[FunctionTool]:
    adk_tools = []
    
    # Get all nodes connected to the tools port
    tool_node_configs = Runtime.get_port_connections(parent_agent_node_id, tools_port_index)
    
    for node_config in tool_node_configs:
        node_type = node_config.get('type', '')  # "Package.SendMail"
        
        # Check if this node has runtime.Tool
        tool = create_tool_from_runtime_tool_node(node_config)
        if tool:
            adk_tools.append(tool)
            
    return adk_tools

def create_tool_from_runtime_tool_node(node_config):
    spec = node_config['config']['config']['spec']
    
    # Extract tool info
    tool_spec = spec.get('tool')  # {"name": "send_email", "description": "..."}
    if not tool_spec:
        return None
        
    tool_name = tool_spec['name']  # "send_email"
    tool_description = tool_spec['description']
    node_guid = node_config['config']['config']['guid']  # "send-mail-node-123"
    
    # Generate JSON schema from spec - ONLY variables with aiScope=true AND scope="AI"
    parameters_schema = generate_json_schema_from_node_spec(spec)
    
    # Register for later tool execution
    register_tool(node_guid, node_config['config']['config'])
    
    # Create ADK FunctionTool
    return _generate_tool_from_json_schema(node_guid, node_type, tool_name, tool_description, parameters_schema)
```

## Step 4: JSON Schema Generation (Python)

```python
def generate_json_schema_from_node_spec(spec):
    schema = {"type": "object", "properties": {}, "required": []}
    
    for prop in spec.get('properties', []):
        for field_name, field_info in prop.get('schema', {}).get('properties', {}).items():
            # CRITICAL: Check if this field has aiScope option AND current scope is set to "AI"
            has_ai_scope_option = field_info.get('aiScope', False)
            current_scope = field_info.get('scope', '')
            
            # Only include if Designer has aiScope option available AND scope is currently set to "AI"
            if not has_ai_scope_option or current_scope != 'AI':
                continue
                
            param_name = field_info.get('name', field_name)  # "to", "body", "subject"
            
            schema['properties'][param_name] = {
                'type': field_info.get('type', 'string'),
                'description': field_info.get('description', '')
            }
            
            # Required if not OptVariable
            if 'Opt' not in field_info.get('variableType', ''):
                schema['required'].append(param_name)
    
    return schema
```

**Generated Schema** (assuming Designer chose scope="AI" for all three parameters):
```json
{
  "type": "object",
  "properties": {
    "to": {
      "type": "string",
      "description": "Email recipient"
    },
    "body": {
      "type": "string", 
      "description": "Email body"
    },
    "subject": {
      "type": "string",
      "description": "Email subject"
    }
  },
  "required": ["to", "body", "subject"]
}
```

**Note**: If Designer changed any parameter's scope from "AI" to "Custom", it would be excluded from the schema.

## Step 5: Function Tool Creation (Python)

```python
def _generate_tool_from_json_schema(node_guid, node_name, tool_name, tool_description, schema):
    # Dynamically generate Python function
    function_code = f"""
def {tool_name}(to: str, body: str, subject: str, tool_context: ToolContext) -> Any:
    '''Send email messages
    
    Args:
        to: Email recipient
        body: Email body
        subject: Email subject
        tool_context: Context for tool execution
    '''
    params = {{'to': to, 'body': body, 'subject': subject}}
    result = callback(params, tool_context)
    return result
    """
    
    # Create callback that handles cross-language communication
    callback = _function_call_simulation(node_guid, tool_name)
    
    # Execute code to create function
    exec(function_code, globals_with_callback)
    generated_function = globals_with_callback[tool_name]
    
    # Return ADK FunctionTool
    return FunctionTool(generated_function)
```

## Step 6: Agent Registration (Python)

```python
# The LLM Agent gets the tools
agent_kwargs = {
    'name': 'My Agent',
    'tools': connect_tools(self.guid, TOOLS_PORT_INDEX),  # Includes our send_email tool
    'model': configured_model
}
agent_instance = Agent(**agent_kwargs)
```

**LLM sees this tool**:
```json
{
  "name": "send_email",
  "description": "Send email messages", 
  "parameters": {
    "type": "object",
    "properties": {
      "to": {"type": "string", "description": "Email recipient"},
      "body": {"type": "string", "description": "Email body"},
      "subject": {"type": "string", "description": "Email subject"}
    },
    "required": ["to", "body", "subject"]
  }
}
```

## Step 7: User Query & LLM Tool Call

**User**: "Send an email to john@company.com saying his order has shipped"

**LLM decides to use tool**:
```json
{
  "tool_calls": [{
    "function": {
      "name": "send_email",
      "arguments": {
        "to": "john@company.com",
        "subject": "Your Order Has Shipped",
        "body": "Good news! Your order has been shipped and is on its way."
      }
    }
  }]
}
```

## Step 8: Cross-Language Tool Execution (Python → Go)

```python
def _function_call_simulation(tool_guid, tool_name):
    def tool_func(parameters=None, tool_context=None):
        session_id = tool_context._invocation_context.session.id
        caller_id = str(uuid.uuid4())  # "call-456-789"
        
        # Get original agent context
        ctx = get_root_agent_message(session_id)
        
        # Create tool request message
        tool_request_ctx = ctx.clone()
        tool_request_ctx.set("__message_type__", "tool_request")
        tool_request_ctx.set("__tool_caller_id__", caller_id)
        tool_request_ctx.set("__tool_name__", tool_name)
        tool_request_ctx.set("__agent_node_id__", parent_agent_node_id)
        
        # Map LLM parameters to context
        if parameters:  # {"to": "john@company.com", "subject": "Your Order...", "body": "Good news!..."}
            for param_name, param_value in parameters.items():
                tool_request_ctx.set(param_name, param_value)
        
        # Send message to Go node via gRPC
        Event.emit_input(tool_guid, tool_request_ctx.get_raw())
        
        # Wait for response
        tool_simulation_wait(caller_id)
        return get_tool_result(caller_id)
    
    return tool_func
```

**Message sent to Go node**:
```json
{
  "__message_type__": "tool_request",  
  "__tool_caller_id__": "call-456-789",
  "__agent_node_id__": "llm-agent-123",
  "to": "john@company.com",
  "subject": "Your Order Has Shipped",
  "body": "Good news! Your order has been shipped and is on its way."
}
```

## Step 9: Go Node Processing (ZERO Additional Code!)

```go
// Package developer writes ZERO tool-specific code!
func (n *SendMailNode) OnMessage(ctx message.Context) error {
    // Just normal processing - works for both RPA and AI tool contexts
    to, _ := n.InTo.Get(ctx)      // Gets from LLM parameters OR previous node
    body, _ := n.InBody.Get(ctx)  // Gets from LLM parameters OR previous node
    subject, _ := n.InSubject.Get(ctx) // Gets from LLM parameters OR Designer config (depends on scope choice)
    
    // Never AI-controllable (no aiScope option)
    server, _ := n.InSMTPServer.Get(ctx)  // Always from Designer config
    
    // Actual email sending logic
    messageID, err := sendEmail(server, from, to, subject, body)
    if err != nil {
        return err // Runtime automatically converts to tool error response
    }
    
    // Set outputs - runtime automatically collects these for tool responses
    n.OutSuccess.Set(ctx, true)
    n.OutMessageID.Set(ctx, messageID)
    
    return nil // Runtime handles tool response automatically
}
```

### What Happens Behind the Scenes (Runtime)

```go
// robomotion-go/runtime/tool_interceptor.go
func (ti *ToolInterceptor) OnMessage(ctx message.Context) error {
    if ti.hasTool && IsToolRequest(ctx) {
        // This is a tool request - handle automatically
        err := ti.originalHandler.OnMessage(ctx) // Call package code normally
        
        if err != nil {
            return ToolResponse(ctx, "error", nil, err.Error())
        } else {
            // Automatically collect output variables and send response
            outputs := ti.collectOutputVariables(ctx)
            return ToolResponse(ctx, "success", outputs, "")
        }
    }
    
    // Normal RPA flow
    return ti.originalHandler.OnMessage(ctx)
}
```

## Step 10: Tool Response (Go → Python)

```go
func ToolResponse(ctx message.Context, status string, data map[string]interface{}, errorMsg string) error {
    callerID, _ := ctx.Get("__tool_caller_id__")     // "call-456-789"
    agentNodeID, _ := ctx.Get("__agent_node_id__")   // "llm-agent-123"
    
    // Create response message
    responseCtx := message.NewContext()
    responseCtx.Set("__message_type__", "tool_response")
    responseCtx.Set("__tool_caller_id__", callerID)
    responseCtx.Set("__tool_status__", status)        // "success"
    responseCtx.Set("__tool_data__", data)            // {"message_id": "msg-789", ...}
    
    // Send back to Python LLM Agent
    if agentID, ok := agentNodeID.(string); ok {
        event.EmitInput(agentID, responseCtx)  // Direct gRPC call to Python
    }
    
    // CRITICAL: Prevent message from flowing to next node in RPA flow
    ctx.SetRaw(nil)
    
    return nil
}
```

**Response message**:
```json
{
  "__message_type__": "tool_response",
  "__tool_caller_id__": "call-456-789",
  "__tool_status__": "success",
  "__tool_data__": {
    "message_id": "msg-789",
    "sent_to": "john@company.com", 
    "timestamp": 1704067200
  }
}
```

## Step 11: Python Agent Receives Response

```python
async def on_message(self, ctx: Context):
    # Check if this is a tool response
    message_type = ctx.get("__message_type__")
    if message_type == "tool_response":
        caller_id = ctx.get("__tool_caller_id__")  # "call-456-789"
        
        if caller_id:
            # Extract tool result
            tool_result = {
                "status": ctx.get("__tool_status__", "success"),
                "error": ctx.get("__tool_error__"),
                "data": ctx.get("__tool_data__", {})
            }
            
            # Notify waiting tool execution
            from ..simulation.tool_simulation import set_tool_result, tool_simulation_end
            set_tool_result(caller_id, tool_result)
            tool_simulation_end(caller_id, tool_result)  # Releases waiting thread
            
            return  # Don't process as normal message
    
    # Normal agent processing...
```

## Step 12: Response Back to LLM

The tool execution completes and returns to the LLM:

```json
{
  "status": "success",
  "data": {
    "message_id": "msg-789",
    "sent_to": "john@company.com",
    "timestamp": 1704067200
  }
}
```

**LLM responds to user**: "I've successfully sent an email to john@company.com letting him know his order has shipped. The email was sent with message ID msg-789."

## Key Technical Innovations

### 1. **Dual Message Flow**
- **Normal RPA**: `Inject → Send Mail → Next Node` (messages flow through)
- **Tool Usage**: `LLM Agent → Send Mail → Response back to LLM` (no forward flow)

### 2. **Designer Choice Model**
- `aiScope` option: Package developer allows AI control possibility
- `scope=AI`: Designer chooses to let LLM control this parameter
- `scope=Custom`: Designer chooses to keep this parameter under Designer control
- Parameters without `aiScope` can never be AI-controlled

### 3. **Zero-Copy Cross-Language Integration**
- Go nodes don't need to know about Python
- Python generates tools dynamically from Go specs
- gRPC handles all cross-language communication

### 4. **Unified Variable System**
```go
// Same code works for both contexts!
to, _ := n.InTo.Get(ctx)  // Gets from LLM params OR Designer config
```

### 5. **Thread-Safe Tool Execution**
- Each tool call gets unique `caller_id`
- Python thread waits for Go response using threading.Event
- Results stored in thread-safe maps

## Communication Protocols

### Message Types
- `tool_request`: Python → Go (tool invocation)
- `tool_response`: Go → Python (tool result)

### Special Context Keys
- `__message_type__`: Identifies the message type
- `__tool_caller_id__`: Unique identifier for each tool call
- `__agent_node_id__`: ID of the LLM Agent that initiated the call
- `__tool_name__`: Name of the tool being called
- `__tool_status__`: Result status (success/error)
- `__tool_data__`: Tool execution results
- `__tool_error__`: Error message if status is error

### gRPC Communication Flow
1. **Tool Discovery**: Python reads Go node specs via gRPC
2. **Tool Execution**: Python sends tool_request to Go via `Event.emit_input`
3. **Tool Response**: Go sends tool_response back via `event.EmitInput`

## Error Handling

### Go Side
```go
if err := processEmail(params); err != nil {
    return runtime.ToolResponse(ctx, "error", nil, err.Error())
}
```

### Python Side
Tools that return error status are handled by the LLM framework and can trigger retry logic or error responses to users.

## Performance Considerations

### Thread Safety
- All tool registries use mutex locks
- Tool results stored in thread-safe maps
- Each tool call has unique caller_id to prevent collisions

### Memory Management
- Tool contexts are cloned to prevent interference
- Results are cleaned up after tool completion
- Large objects use the LMO (Large Message Object) system

### Scalability
- Multiple tools can execute concurrently
- Each tool call is independent
- gRPC handles connection pooling and load balancing

## The Magic: Zero Changes Needed

The existing Send Mail Go node works **unchanged** for both:
1. **Traditional RPA flows**: Designer connects nodes visually
2. **AI tool usage**: LLM calls functions dynamically

Just adding `runtime.Tool` and `aiScope` options makes any RPA node AI-ready! 🎯

## Security Considerations

### Parameter Isolation  
- LLM can only control parameters where Designer chose `scope=AI`
- Parameters without `aiScope` option can never be AI-controlled
- Designer maintains full control over what AI can access
- No direct access to sensitive configuration

### Message Validation
- All cross-language messages are validated
- Tool requests must come from registered agents
- Response routing uses secure node IDs

### Error Information
- Error messages are sanitized before sending to LLM
- No internal system information exposed
- Stack traces and sensitive data filtered

This technical architecture enables seamless cross-language AI tool integration while maintaining security, performance, and the existing RPA development workflow.