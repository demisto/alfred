package web

import (
	"testing"
)

func TestIsBanned(t *testing.T) {
	bannedSamples := []string{"2.144.0.0:0", "2.147.100.100:0", "2.147.255.255:0", "5.112.0.0:0", "5.127.255.255:0"}
	cleanSamples := []string{"1.1.1.1:0", "73.231.0.156:0", "104.197.111.48:0"}

	for _, ip := range bannedSamples {
		if !isBanned(ip) {
			t.Errorf("IP %s was not banned\n", ip)
		}
	}
	for _, ip := range cleanSamples {
		if isBanned(ip) {
			t.Errorf("IP %s was banned\n", ip)
		}
	}
}
