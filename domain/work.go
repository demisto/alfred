package domain

import (
	"errors"
	"time"

	"github.com/demisto/alfred/slack"
	"github.com/demisto/goxforce"
	"github.com/demisto/infinigo"
	"github.com/slavikm/govt"
)

// Context to push with each message to identify the relevant team and user
type Context struct {
	Team         string `json:"team"`
	User         string `json:"user"`
	OriginalUser string `json:"original_user"`
	Channel      string `json:"channel"`
	Type         string `json:"type"`
}

// contextFromMap ...
func contextFromMap(c map[string]interface{}) *Context {
	return &Context{
		Team:         c["team"].(string),
		User:         c["user"].(string),
		OriginalUser: c["original_user"].(string),
		Channel:      c["channel"].(string),
		Type:         c["type"].(string),
	}
}

// GetContext from a message based on actual type
func GetContext(context interface{}) (*Context, error) {
	switch c := context.(type) {
	case *Context:
		// Hack to duplicate the context so if we are using channels not to override it
		cx := *c
		return &cx, nil
	case map[string]interface{}:
		return contextFromMap(c), nil
	default:
		return nil, errors.New("Unknown context")
	}
}

// File details for a request
type File struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Name  string `json:"name"`
	Size  int    `json:"size"`
	Token string `json:"token"`
}

// WorkRequest contains the relevant fields for a work request
type WorkRequest struct {
	MessageID  string      `json:"message_id"`
	Type       string      `json:"type"`
	Text       string      `json:"text"`
	File       File        `json:"file"`
	ReplyQueue string      `json:"reply_queue"`
	Context    interface{} `json:"context"`
	Online     bool        `json:"online"`   // Are we running this request from online details page
	VTKey      string      `json:"vt_key"`   // This team has his own vt key
	XFEKey     string      `json:"xfe_key"`  // This team has his own xfe key
	XFEPass    string      `json:"xfe_pass"` // This team has his own xfe pass
}

// WorkRequestFromMessage converts a message to a work request
func WorkRequestFromMessage(msg slack.Response, token, vtKey, xfeKey, xfePass string) *WorkRequest {
	req := &WorkRequest{VTKey: vtKey, XFEKey: xfeKey, XFEPass: xfePass}
	switch msg.S("type") {
	case "message":
		switch msg.S("subtype") {
		case "":
			req.MessageID, req.Type, req.Text = msg.S("ts"), "message", msg.S("text")
		case "message_changed":
			req.MessageID, req.Type, req.Text = msg.S("message.ts"), "message", msg.S("message.text")
		case "file_share", "file_mention":
			req.MessageID, req.Type, req.File = msg.S("ts"), "file", File{ID: msg.S("file.id"), URL: msg.S("file.url_private"), Name: msg.S("file.name"), Size: msg.I("file.size"), Token: token}
		case "file_comment":
			req.MessageID, req.Type, req.Text = msg.S("ts"), "message", msg.S("comment.comment")
		}
	// If this message is file upload and we got it (meaning the user is ours)
	case "file_created":
		req.Type, req.File = "file", File{ID: msg.S("file.id"), URL: msg.S("file.url"), Name: msg.S("file.name"), Size: msg.I("file.size")}
	}
	return req
}

const (
	// ReplyTypeHash for hash replies
	ReplyTypeHash int = 1 << iota
	// ReplyTypeURL for URL replies
	ReplyTypeURL
	// ReplyTypeIP for IP replies
	ReplyTypeIP
	// ReplyTypeFile for File replies
	ReplyTypeFile
)

const (
	// ResultClean from the scan if it is not known bad and at least one service found it to be clean
	ResultClean int = iota
	// ResultDirty if at least one service convicted it
	ResultDirty
	// ResultUnknown if none of the services knowns about the request
	ResultUnknown
)

// HashReply holds the information about a hash
type HashReply struct {
	Details string `json:"details"`
	Result  int
	XFE     struct {
		NotFound bool             `json:"not_found"`
		Error    string           `json:"error"`
		Malware  goxforce.Malware `json:"malware"`
	} `json:"xfe"`
	VT struct {
		Error      string          `json:"error"`
		FileReport govt.FileReport `json:"file_report"`
	} `json:"vt"`
	Cy struct {
		Error  string                 `json:"error"`
		Result infinigo.QueryResponse `json:"result"`
	} `json:"cy"`
}

// URLReply holds the information about a URL
type URLReply struct {
	Details string `json:"details"`
	Result  int
	XFE     struct {
		NotFound   bool                    `json:"not_found"`
		Error      string                  `json:"error"`
		Resolve    goxforce.ResolveResp    `json:"resolve"`
		URLDetails goxforce.URL            `json:"url_details"`
		URLMalware goxforce.URLMalwareResp `json:"url_malware"`
	} `json:"xfe"`
	VT struct {
		Error     string         `json:"error"`
		URLReport govt.UrlReport `json:"url_report"`
	} `json:"vt"`
}

// IPReply holds the information about an IP
type IPReply struct {
	Details string `json:"details"`
	Result  int
	Private bool `json:"private"`
	XFE     struct {
		NotFound     bool                  `json:"not_found"`
		Error        string                `json:"error"`
		IPReputation goxforce.IPReputation `json:"ip_reputation"`
		IPHistory    goxforce.IPHistory    `json:"ip_history"`
	} `json:"xfe"`
	VT struct {
		Error    string        `json:"error"`
		IPReport govt.IpReport `json:"ip_report"`
	} `json:"vt"`
}

// FileReply holds the information about a File
type FileReply struct {
	Result       int
	FileTooLarge bool   `json:"file_too_large"`
	Virus        string `json:"virus"`
	Error        string `json:"error"`
	Details      File   `json:"details"`
}

// WorkReply to a work request being done
type WorkReply struct {
	Type      int         `json:"type"`
	MessageID string      `json:"message_id"`
	Hashes    []HashReply `json:"hashes"`
	URLs      []URLReply  `json:"urls"`
	IPs       []IPReply   `json:"ips"`
	File      FileReply   `json:"file"`
	Context   interface{} `json:"context"`
}

// MaliciousContent holds info about convicted content
type MaliciousContent struct {
	Team        string `json:"team"`
	Channel     string `json:"channel"`
	MessageID   string `json:"message_id" db:"message_id"`
	ContentType int    `json:"content_type" db:"content_type"`
	Content     string `json:"content"`
	FileName    string `json:"file_name" db:"file_name"`
	VT          string `json:"vt"`
	XFE         string `json:"xfe"`
	Cy          string `json:"cy"`
	ClamAV      string `json:"clamav"`
}

// UniqueID of the message
func (mc *MaliciousContent) UniqueID() string {
	return mc.Team + "," + mc.Channel + "," + mc.MessageID
}

// DBQueueMessage holds a message passed via the database
type DBQueueMessage struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	MessageType string    `json:"message_type" db:"message_type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"ts" db:"ts"`
}
