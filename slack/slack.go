package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sirupsen/logrus"
)

// client to the Slack API.
type Client struct {
	Token string // The token to use for requests. Required.
}

// Response to Slack web-api calls
type Response map[string]interface{}

// Get a recursive path
func (r Response) Get(path string) interface{} {
	parts := strings.Split(path, ".")
	curr := map[string]interface{}(r)
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

// R returns a path as a response
func (r Response) R(path string) Response {
	if d := r.Get(path); d != nil {
		if dr, ok := d.(map[string]interface{}); ok {
			return Response(dr)
		}
	}
	return Response(map[string]interface{}{})
}

// S returns given path as string
func (r Response) S(path string) string {
	if d := r.Get(path); d != nil {
		if ds, ok := d.(string); ok {
			return ds
		}
	}
	return ""
}

// B returns given path as bool
func (r Response) B(path string) bool {
	if d := r.Get(path); d != nil {
		if db, ok := d.(bool); ok {
			return db
		}
	}
	return false
}

// I returns given path as int
func (r Response) I(path string) int {
	if d := r.Get(path); d != nil {
		if db, ok := d.(int); ok {
			return db
		}
	}
	return 0
}

// OK returns true if response is ok
func (r Response) OK() bool {
	return r.B("ok")
}

// Error returns an error of the response
func (r Response) Error() string {
	return r.S("error")
}

// Warning returns warning if the response contains warnings
func (r Response) Warning() string {
	return r.S("warning")
}

// Do the given API request
// Returns the response if the status code is between 200 and 299
func (s *Client) Do(method, path string, body interface{}) (Response, error) {
	var bodyReader io.Reader
	if method == "GET" {
		if body != nil {
			if bmap, ok := body.(map[string]string); ok {
				urlValues := url.Values{}
				for k, v := range bmap {
					urlValues.Set(k, v)
				}
				path += "?" + urlValues.Encode()
			}
		}
	} else {
		if body != nil {
			b, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(b)
		}
	}
	req, err := http.NewRequest(method, "https://slack.com/api/"+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if method != "GET" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	req.Header.Set("Accept", "application/json")
	if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New("unexpected status code: [" + resp.Status + "]")
	}
	res := Response{}
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, errors.New("Slack error: " + res.Error())
	}
	if res.Warning() != "" {
		logrus.Warnf("Slack API warning %s", res.Warning())
	}
	return res, nil
}
