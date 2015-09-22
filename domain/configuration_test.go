package domain

import "testing"

// TestRandomEvents tests the generation of random events
func TestIsActive(t *testing.T) {
	var c Configuration
	if c.IsActive() {
		t.Error("IsActive is true for empty configuration")
	}

	c.Channels = []string{"kuku"}
	if !c.IsActive() {
		t.Error("IsActive is false but we have a channel")
	}

	c.Channels = []string{}
	c.Groups = []string{"kuku"}
	if !c.IsActive() {
		t.Error("IsActive is false but we have a group")
	}

	c.Groups = []string{}
	c.IM = true
	if !c.IsActive() {
		t.Error("IsActive is false but we have an IM")
	}
}

func TestIsInterestedIn(t *testing.T) {
	var c Configuration
	if c.IsInterestedIn("Cx", "") || c.IsInterestedIn("Gx", "") || c.IsInterestedIn("Dx", "") {
		t.Error("Configuration is empty but still interested")
	}

	c.Channels = []string{"Cx"}
	c.Groups = []string{"Gx"}
	c.IM = true

	if !c.IsInterestedIn("Cx", "") || !c.IsInterestedIn("Gx", "") || !c.IsInterestedIn("Dx", "") {
		t.Error("Configuration is not interested but it should")
	}
}
