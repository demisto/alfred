package domain

import (
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/util"
)

// Configuration holds the user configuration
type Configuration struct {
	Channels []string `json:"channels"`
	Groups   []string `json:"groups"`
	IM       bool     `json:"im"`
	Regexp   string   `json:"regexp"`
	All      bool     `json:"all"`
}

// IsActive returns true if there is at least one active part for the user
func (c *Configuration) IsActive() bool {
	return c.All || len(c.Channels) > 0 || len(c.Groups) > 0 || c.IM
}

// IsInterestedIn the given channel
func (c *Configuration) IsInterestedIn(channel, channelName string) bool {
	logrus.Debugf("Checking interest in %s, %s for %+v\n", channel, channelName, c)
	if len(channel) == 0 {
		return false
	}
	if c.All {
		return true
	}
	found := false
	switch channel[0] {
	case 'C':
		found = util.In(c.Channels, channel)
	case 'G':
		found = util.In(c.Groups, channel)
	case 'D':
		return c.IM
	}
	if !found && c.Regexp != "" {
		re, err := regexp.Compile(c.Regexp)
		if err != nil {
			logrus.Warnf("Found invalid regexp in configuration - %v\n", err)
		} else {
			logrus.Debugf("Matching %s\n", c.Regexp)
			return re.MatchString(channelName)
		}
	}
	return found
}
