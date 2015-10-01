package util

import (
	"fmt"
	"math/rand"
	"net/url"
	"reflect"
	"strings"
)

// Max returns the max between 2 ints
func Max(a int, b int) int {
	if a < b {
		return b
	}
	return a
}

// Min returns the min between 2 ints
func Min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}

// ToIntf converts a slice or array of a specific type to array of interface{}
func ToIntf(s interface{}) []interface{} {
	v := reflect.ValueOf(s)
	// There is no need to check, we want to panic if it's not slice or array
	intf := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		intf[i] = v.Index(i).Interface()
	}
	return intf
}

// RandStr returns a random string of size strSize
func RandStr(strSize int) string {
	dictionary := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, strSize)
	for k := range bytes {
		v := rand.Int()
		bytes[k] = dictionary[v%len(dictionary)]
	}
	return string(bytes)
}

// ToLower conerts a slice of strings to lower case
func ToLower(s []string) []string {
	res := make([]string, len(s))
	for i, v := range s {
		res[i] = strings.ToLower(v)
	}
	return res
}

// In checks if val is in s slice
func In(s interface{}, val interface{}) bool {
	si := ToIntf(s)
	for _, v := range si {
		if v == val {
			return true
		}
	}
	return false
}

// Canonicalize the given URLs
func Canonicalize(rawurls ...string) []string {
	var canonicalized []string
	for _, rawurl := range rawurls {
		u, err := url.Parse(rawurl)
		if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			u.Fragment = ""
			u.Path = ""
			u.RawQuery = ""
			canonicalized = append(canonicalized, u.String())
		}
	}
	return canonicalized
}

// Substr of string based on runes and not bytes
func Substr(s string, from int, to int) string {
	if from < 0 || to < from {
		panic(fmt.Sprintf("Must specify valid from and to [%d %d]", from, to))
	}
	r := []rune(s)
	l := len(r)
	if from >= l {
		return ""
	}
	if to > l {
		to = l
	}
	return string(r[from:to])
}
