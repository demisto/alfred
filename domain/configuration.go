package domain

// Configuration holds the user configuration
type Configuration struct {
	Channels []string `json:"channels"`
	Groups   []string `json:"groups"`
	IM       bool     `json:"im"`
}
