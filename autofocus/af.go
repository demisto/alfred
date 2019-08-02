package autofocus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/util"
)

// URL of AutoFocus API
const URL = "https://autofocus.paloaltonetworks.com/api/v1.0"

// Client to the AutoFocus API
type Client struct {
	Token string // The token to use for requests. Required.
}

// Reputation of given hash
type Reputation struct {
	Malware   bool      `json:"malware"`    // Is this known bad
	Tags      []string  `json:"tags"`       // Tags associated with this malware
	TagGroups []string  `json:"tag_groups"` // TagGroups relevant to this malware
	FileType  string    `json:"file_type"`  // Type of the file
	Created   time.Time `json:"created"`    // When was this create
	Regions   []string  `json:"regions"`    // Regions where this was seen
}

func (rep *Reputation) String() string {
	if !rep.Malware {
		if rep.Created.IsZero() {
			return "Unknown sample"
		}
		return fmt.Sprintf("Clean sample created %v", rep.Created)
	}
	return fmt.Sprintf("Malicious %s sample created %v in regions [%s] with groups [%s] and tags [%s]",
		rep.FileType, rep.Created, strings.Join(rep.Regions, ", "), strings.Join(rep.TagGroups, ", "),
		strings.Join(rep.Tags, ", "))
}

// Do the given API request
// Returns the response if the status code is between 200 and 299
func (c *Client) Do(path string, body map[string]interface{}) (util.Object, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("must provide af key")
	}
	if body == nil {
		body = make(map[string]interface{})
	}
	body["apiKey"] = c.Token
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(URL+"/"+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("unexpected status code: [%d] [%s]", resp.StatusCode, resp.Status)
	}
	res := util.Object{}
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res, nil
}

// HashReputation returns the reputation of given hash
func (c *Client) HashReputation(hash string) *Reputation {
	if c.Token == "" {
		return nil
	}
	hashType := "md5"
	if len(hash) == 64 {
		hashType = "sha256"
	} else if len(hash) == 40 {
		hashType = "sha1"
	}
	query := make(map[string]interface{})
	_ = json.Unmarshal([]byte(fmt.Sprintf(`{"operator": "all", "children": [{"field": "sample.%s", "operator": "is", "value": "%s"}]}`, hashType, hash)), &query)
	args := map[string]interface{}{"scope": "public", "size": 1, "from": 0, "query": query}
	res, err := c.Do("samples/search/", args)
	if err != nil {
		logrus.WithError(err).Infof("error executing AF search")
		return nil
	}
	cookie := res.S("af_cookie")
	if cookie == "" {
		logrus.Infof("error executing AF search - did not receive cookie")
		return nil
	}
	// Try every 10 seconds for 5 times
	found := false
	for i := 0; i < 5 && !found; i++ {
		time.Sleep(time.Second * 10)
		res, err := c.Do("samples/results/"+cookie, nil)
		if err != nil {
			logrus.WithError(err).Infof("error executing AF search results")
			return nil
		}
		hits := res.A("hits")
		if res.S("af_message") == "complete" || !res.B("af_in_progress") || len(hits) > 0 {
			if len(hits) > 0 {
				if obj, ok := hits[0].(map[string]interface{}); ok {
					hit := util.Object(obj)
					hit = hit.O("_source")
					reputation := &Reputation{}
					reputation.Malware = hit.I("malware") == 1
					reputation.FileType = hit.S("filetype")
					reputation.Regions = hit.AStr("region")
					reputation.Tags = hit.AStr("tag")
					reputation.TagGroups = hit.AStr("tag_groups")
					createdStr := hit.S("create_date")
					if created, err := time.Parse("2006-01-02T15:04:05", createdStr); err == nil {
						reputation.Created = created
					} else {
						logrus.WithError(err).Infof("error converting AF timestamp")
					}
					return reputation
				}
			}
			return nil // It was complete but there are no hits
		}
	}
	return nil
}
