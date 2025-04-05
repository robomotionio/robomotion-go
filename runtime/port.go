package runtime

type Port struct {
	Direction          string   `json:"direction"`
	Position           string   `json:"position"`
	Name               string   `json:"name"`
	AllowedConnections []string `json:"allowedConnections,omitempty"`
	MaxConnections     int      `json:"maxConnections,omitempty"`
	Icon               string   `json:"icon"`
	Color              string   `json:"color"`
}
