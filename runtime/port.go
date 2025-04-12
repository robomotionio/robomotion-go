package runtime

// Port is a type alias for []string that stores the GUIDs of connected nodes
type Port []string

// NodeInfo contains information about a connected node
type NodeInfo struct {
	Type    string                 `json:"type"`    // Type of the node
	Version string                 `json:"version"` // Version of the node
	Config  map[string]interface{} `json:"config"`  // The node's configuration data (JSON bytes)
}
