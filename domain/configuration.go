package domain

import "github.com/demisto/server/util"

// Configuration holds the user configuration
type Configuration struct {
	Channels []string `json:"channels"`
	Groups   []string `json:"groups"`
	IM       bool     `json:"im"`
}

// IsActive returns true if there is at least one active part for the user
func (c *Configuration) IsActive() bool {
	return len(c.Channels) > 0 || len(c.Groups) > 0 || c.IM
}

// IsInterestedIn the given channel
func (c *Configuration) IsInterestedIn(channel string) bool {
	if len(channel) == 0 {
		return false
	}
	switch channel[0] {
	case 'C':
		return util.In(c.Channels, channel)
	case 'G':
		return util.In(c.Groups, channel)
	case 'D':
		return c.IM
	}
	return false
}
