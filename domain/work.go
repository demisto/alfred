package domain

import (
	"github.com/demisto/goxforce"
	"github.com/demisto/slack"
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

// File details for a request
type File struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
	Size int    `json:"size"`
}

// WorkRequest contains the relevant fields for a work request
type WorkRequest struct {
	MessageID  string      `json:"message_id"`
	Type       string      `json:"type"`
	Text       string      `json:"text"`
	File       File        `json:"file"`
	ReplyQueue string      `json:"reply_queue"`
	Context    interface{} `json:"context"`
	Online     bool        `json:"online"` // Are we running this request from online details page
}

// WorkRequestFromMessage converts a message to a work request
func WorkRequestFromMessage(msg *slack.Message) *WorkRequest {
	req := &WorkRequest{}
	switch msg.Type {
	case "message":
		switch msg.Subtype {
		case "":
			req.MessageID, req.Type, req.Text = msg.Timestamp, "message", msg.Text
		case "message_changed":
			req.MessageID, req.Type, req.Text = msg.Message.Timestamp, "message", msg.Message.Text
		case "file_share", "file_mention":
			req.MessageID, req.Type, req.File = msg.Timestamp, "file", File{ID: msg.File.ID, URL: msg.File.URL, Name: msg.File.Name, Size: msg.File.Size}
		case "file_comment":
			req.MessageID, req.Type, req.Text = msg.Timestamp, "message", msg.Comment.Comment
		}
	// If this message is file upload and we got it (meaning the user is ours)
	case "file_created":
		req.Type, req.File = "file", File{ID: msg.File.ID, URL: msg.File.URL, Name: msg.File.Name, Size: msg.File.Size}
	}
	return req
}

const (
	// ReplyTypeMD5 for MD5 replies
	ReplyTypeMD5 int = 1 << iota
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

// WorkReply to a work request being done
type WorkReply struct {
	Type      int    `json:"type"`
	MessageID string `json:"message_id"`
	MD5       struct {
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
	} `json:"md5"`
	URL struct {
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
	} `json:"url"`
	IP struct {
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
	} `json:"ip"`
	File struct {
		Result       int
		FileTooLarge bool   `json:"file_too_large"`
		Virus        string `json:"virus"`
		Error        string `json:"error"`
		Details      File   `json:"details"`
	} `json:"file"`
	Context interface{} `json:"context"`
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
	ClamAV      string `json:"clamav"`
}

// UniqueID of the message
func (mc *MaliciousContent) UniqueID() string {
	return mc.Team + "," + mc.Channel + "," + mc.MessageID
}
