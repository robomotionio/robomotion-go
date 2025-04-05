package runtime

// Port is a type alias for []string that stores the GUIDs of connected nodes
type Port []string

// NodeInfo contains information about a connected node
type NodeInfo struct {
	ID      string `json:"id"`      // Node GUID
	Type    string `json:"type"`    // Node type (e.g., "Robomotion.Agent.Tool")
	Name    string `json:"name"`    // Node name
	Version string `json:"version"` // Node version
}

// GetConnectedNodes queries the node GUIDs stored in the Port
// and returns an array of NodeInfo objects
func (p Port) GetConnectedNodes() []NodeInfo {
	// Initialize the result array
	result := make([]NodeInfo, 0, len(p))

	// Iterate through the GUIDs stored in the Port
	for _, guid := range p {
		// In a real implementation, you would query node information
		// from a registry or service using the GUID
		// For now, we'll create a placeholder with just the ID
		info := NodeInfo{
			ID: guid,
			// Type, Name, and Version would be populated from the registry
		}

		result = append(result, info)
	}

	return result
}
