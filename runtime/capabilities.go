package runtime

var (
	capabilities = map[string]interface{}{
		"optimize-large-object": true,
	}
)

func GetCapabilities() map[string]interface{} {
	return capabilities
}

func GetCapability(name string) interface{} {
	return capabilities[name]
}
