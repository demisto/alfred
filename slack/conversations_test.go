package slack

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClient_Conversations(t *testing.T) {
	s := Client{Token: "xoxp-3435591503-3435591511-6060683735-fcce7d"}
	conversations, err := s.Conversations("")
	assert.NoError(t, err)
	found := false
	for _, conversation := range conversations {
		if conversation.S("name") == "general" {
			found = true
			break
		}
	}
	assert.True(t, found)
}
