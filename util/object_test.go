package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObject_Get(t *testing.T) {
	r := Object(map[string]interface{}{"a": 1, "b": "2", "c": map[string]interface{}{"x": "11", "y": map[string]interface{}{"z": "111"}}})
	assert.Equal(t, 1, r.Get("a"))
	assert.Equal(t, "2", r.Get("b"))
	assert.Nil(t, r.Get("xxx"))
	assert.Equal(t, "11", r.Get("c.x"))
	assert.Equal(t, "111", r.Get("c.y.z"))
	r1 := r.O("c.y")
	assert.Equal(t, "111", r1.Get("z"))
}
