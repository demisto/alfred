package slack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_Conversations(t *testing.T) {
	s := Client{Token: ""}
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
