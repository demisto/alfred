package slack

// Conversations retrieval by type
// Handle cursors as well
func (s *Client) Conversations(t string) (channels []Response, err error) {
	args := map[string]string{"exclude_archived": "true", "limit": "1000"}
	if t != "" {
		args["types"] = t
	}
	channels = make([]Response, 0)
	for {
		res, err := s.Do("GET", "conversations.list", args)
		if err != nil {
			return nil, err
		}
		if c, ok := res["channels"]; ok {
			for _, cc := range c.([]interface{}) {
				channels = append(channels, Response(cc.(map[string]interface{})))
			}
		}
		if res.S("response_metadata.next_cursor") == "" {
			break
		} else {
			args["cursor"] = res.S("response_metadata.next_cursor")
		}
	}
	return
}
