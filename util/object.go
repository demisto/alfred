package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Object generic map to allow JSON nice interactivity
type Object map[string]interface{}

// Get a recursive path
func (o Object) Get(path string) interface{} {
	parts := strings.Split(path, ".")
	curr := map[string]interface{}(o)
	for i, p := range parts {
		if tmp, ok := curr[p]; ok {
			if i == len(parts)-1 {
				return tmp
			}
			curr, ok = tmp.(map[string]interface{})
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}
	return nil
}

// O returns a path as an Object
func (o Object) O(path string) Object {
	if d := o.Get(path); d != nil {
		if dr, ok := d.(map[string]interface{}); ok {
			return Object(dr)
		}
	}
	return Object(map[string]interface{}{})
}

// S returns given path as string
func (o Object) S(path string) string {
	if d := o.Get(path); d != nil {
		if ds, ok := d.(string); ok {
			return ds
		}
	}
	return ""
}

// B returns given path as bool
func (o Object) B(path string) bool {
	if d := o.Get(path); d != nil {
		if db, ok := d.(bool); ok {
			return db
		}
	}
	return false
}

// I returns given path as int
func (o Object) I(path string) int {
	if d := o.Get(path); d != nil {
		if di, ok := d.(int); ok {
			return di
		}
		if di, ok := d.(float64); ok {
			return int(di)
		}
	}
	return 0
}

// A returns given path as []interface{}
func (o Object) A(path string) []interface{} {
	if d := o.Get(path); d != nil {
		if da, ok := d.([]interface{}); ok {
			return da
		}
	}
	return nil
}

// AStr returns given path as []string
func (o Object) AStr(path string) []string {
	if d := o.Get(path); d != nil {
		if da, ok := d.([]interface{}); ok {
			dastr := make([]string, len(da))
			for i, elm := range da {
				dastr[i] = fmt.Sprintf("%v", elm)
			}
			return dastr
		}
	}
	return nil
}

// Stringify the map to JSON string
func (o Object) Stringify() string {
	return ToJSONString(o)
}

// NewObject by parsing the given byte data
func NewObject(b []byte) (Object, error) {
	o := make(map[string]interface{})
	err := json.Unmarshal(b, &o)
	return Object(o), err
}
