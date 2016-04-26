package domain

import (
	"time"

	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/util"
)

// UserType is the type of the user Hipchat or Slack
type UserType int

const (
	// UserTypeSlack is a Slack user
	UserTypeSlack = iota
	// UserTypeHipchat is a Hipchat user
	UserTypeHipchat
)

// Stringer implementation
func (s UserType) String() string {
	switch s {
	case UserTypeSlack:
		return "Slack User"
	case UserTypeHipchat:
		return "Hipchat User"
	default:
		return "Unknown"
	}
}

// UserStatus is the status of the user
type UserStatus int

const (
	// UserStatusActive is an active user
	UserStatusActive = iota
	// UserStatusDeleted is a deleted user
	UserStatusDeleted
)

// Stringer implementation
func (s UserStatus) String() string {
	switch s {
	case UserStatusActive:
		return "Active"
	case UserStatusDeleted:
		return "Deleted"
	default:
		return "Unknown"
	}
}

// User contains all the information of a user
type User struct {
	ID                string     `json:"id"`
	Team              string     `json:"team"`
	Name              string     `json:"name"`
	Type              UserType   `json:"type"`
	Status            UserStatus `json:"status"`
	RealName          string     `json:"real_name" db:"real_name"`
	Email             string     `json:"email"`
	IsBot             bool       `json:"is_bot" db:"is_bot"`
	IsAdmin           bool       `json:"is_admin" db:"is_admin"`
	IsOwner           bool       `json:"is_owner" db:"is_owner"`
	IsPrimaryOwner    bool       `json:"is_primary_owner" db:"is_primary_owner"`
	IsRestricted      bool       `json:"is_restricted" db:"is_restricted"`
	IsUltraRestricted bool       `json:"is_ultra_restricted" db:"is_ultra_restricted"`
	ExternalID        string     `json:"external_id" db:"external_id"`
	Token             string     `json:"token"`
	Created           time.Time  `json:"created"`
}

// ClearToken is returned from the encrypted token
func (u *User) ClearToken() (string, error) {
	if u.Token != "" {
		return util.Decrypt(u.Token, conf.Options.Security.DBKey)
	}
	return "", nil
}

// SecureToken is returned from the clear token
func (u *User) SecureToken() (string, error) {
	if u.Token != "" {
		return util.Encrypt(u.Token, conf.Options.Security.DBKey)
	}
	return "", nil
}

// Team holds information about the team
type Team struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      UserStatus `json:"status"`
	EmailDomain string     `json:"email_domain" db:"email_domain"`
	Domain      string     `json:"domain"`
	Plan        string     `json:"plan"`
	ExternalID  string     `json:"external_id" db:"external_id"`
	Created     time.Time  `json:"created"`
	BotUserID   string     `json:"bot_user_id" db:"bot_user_id"`
	BotToken    string     `json:"bot_token" db:"bot_token"`
	VTKey       string     `json:"vt_key" db:"vt_key"`
	XFEKey      string     `json:"xfe_key" db:"xfe_key"`
	XFEPass     string     `json:"xfe_pass" db:"xfe_pass"`
}

// ClearToken is returned from the encrypted token
func (t *Team) ClearToken() (string, error) {
	if t.BotToken != "" {
		return util.Decrypt(t.BotToken, conf.Options.Security.DBKey)
	}
	return "", nil
}

// ClearVTKey is returned from the encrypted vt key
func (t *Team) ClearVTKey() (string, error) {
	if t.VTKey != "" {
		return util.Decrypt(t.VTKey, conf.Options.Security.DBKey)
	}
	return "", nil
}

// ClearXFEKey is returned from the encrypted xfe key
func (t *Team) ClearXFEKey() (string, error) {
	if t.XFEKey != "" {
		return util.Decrypt(t.XFEKey, conf.Options.Security.DBKey)
	}
	return "", nil
}

// ClearXFEPass is returned from the encrypted xfe pass
func (t *Team) ClearXFEPass() (string, error) {
	if t.XFEPass != "" {
		return util.Decrypt(t.XFEPass, conf.Options.Security.DBKey)
	}
	return "", nil
}

// SecureToken is returned from the clear token
func (t *Team) SecureToken() (string, error) {
	if t.BotToken != "" {
		return util.Encrypt(t.BotToken, conf.Options.Security.DBKey)
	}
	return "", nil
}

// SecureVTKey is returned from the clear vt key
func (t *Team) SecureVTKey() (string, error) {
	if t.VTKey != "" {
		return util.Encrypt(t.VTKey, conf.Options.Security.DBKey)
	}
	return "", nil
}

// SecureXFEKey is returned from the clear xfe key
func (t *Team) SecureXFEKey() (string, error) {
	if t.XFEKey != "" {
		return util.Encrypt(t.XFEKey, conf.Options.Security.DBKey)
	}
	return "", nil
}

// SecureXFEPass is returned from the clear xfe pass
func (t *Team) SecureXFEPass() (string, error) {
	if t.XFEPass != "" {
		return util.Encrypt(t.XFEPass, conf.Options.Security.DBKey)
	}
	return "", nil
}

// OAuthState holds oauth validation state
type OAuthState struct {
	State     string    `json:"state"`
	Timestamp time.Time `json:"ts" db:"ts"`
}

// TeamBot holds allocation of bot for team
type TeamBot struct {
	Team      string    `json:"team"`
	Bot       string    `json:"bot"`
	Timestamp time.Time `json:"ts" db:"ts"`
}

// JoinSlack holds invite information to join our Slack channel
type JoinSlack struct {
	Email     string    `json:"email"`
	Timestamp time.Time `json:"ts" db:"ts"`
	Invited   bool      `json:"invited"`
}
