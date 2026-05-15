package runtime

// Tool is a marker struct that indicates a node can be used as an AI tool.
// Spec generation reads the `tool:"name=...,description=..."` tag on the
// embedding struct's Tool field and emits a single tool entry. The runtime
// ToolInterceptor (tool_interceptor.go) routes incoming tool_request
// messages to the node's OnMessage and auto-responds via ToolResponse.
type Tool struct{}

// Toolkit is the multi-tool counterpart of Tool: a node that embeds Toolkit
// publishes a set of named tools instead of one. Spec generation calls the
// node's Tools() method (ToolkitProvider) and emits a tools[] array. The
// agent that consumes the toolkit registers each tool independently, all
// dispatched to the same node guid; the in-band __tool_name__ discriminator
// on the message context tells the node's OnMessage which tool was called.
//
// A Toolkit node should also implement ToolkitProvider. Optionally, it can
// implement SkillProvider to ship a markdown system-message addendum that
// describes how to use the toolkit (the toolkit's AGENT.md).
type Toolkit struct{}

// ToolDef describes one tool inside a Toolkit. Schema is JSON Schema as a
// map[string]interface{} so authors can build it in code or load it from
// an embedded resource. Required is a list of property names that must be
// supplied — the same shape every modern function-calling API expects.
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
}

// ToolkitProvider is the optional interface a Toolkit node implements to
// publish its tool definitions. The spec generator calls Tools() once at
// package build time. Returning an empty slice is valid (the toolkit
// publishes no tools and is effectively inert).
type ToolkitProvider interface {
	Tools() []ToolDef
}

// SkillProvider is the optional interface any node (toolkit or single
// tool) implements to ship a markdown system-message addendum. The
// consuming agent (Hermes, ADK, …) appends this content to the agent's
// system message when the node is wired to its tools port. Conceptually
// the in-tree equivalent of MCP's prompts/get — the behavioral contract
// travels with the tools.
type SkillProvider interface {
	Skill() string
}
