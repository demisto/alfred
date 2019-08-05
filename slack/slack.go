package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/util"
)

// client to the Slack API.
type Client struct {
	Token string // The token to use for requests. Required.
}

// OK returns true if response is ok
func OK(r util.Object) bool {
	return r.B("ok")
}

// Error returns an error of the response
func Error(r util.Object) string {
	return r.S("error")
}

// Warning returns warning if the response contains warnings
func Warning(r util.Object) string {
	return r.S("warning")
}

// Do the given API request
// Returns the response if the status code is between 200 and 299
func (s *Client) Do(method, path string, body interface{}) (util.Object, error) {
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
	res := util.Object{}
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if !OK(res) {
		return nil, errors.New("Slack error: " + Error(res))
	}
	if Warning(res) != "" {
		logrus.Warnf("Slack API warning %s", Warning(res))
	}
	return res, nil
}
