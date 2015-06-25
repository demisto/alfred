package domain

import "time"

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
	RealName          string     `json:"real_name"`
	Email             string     `json:"email"`
	IsBot             bool       `json:"is_bot"`
	IsAdmin           bool       `json:"is_admin"`
	IsOwner           bool       `json:"is_owner"`
	IsPrimaryOwner    bool       `json:"is_primary_owner"`
	IsRestricted      bool       `json:"is_restricted"`
	IsUltraRestricted bool       `json:"is_ultra_restricted"`
	ExternalID        string     `json:"external_id"`
	Token             string     `json:"token"`
}

// Team holds information about the team
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	EmailDomain string `json:"email_domain"`
	Domain      string `json:"domain"`
	Plan        string `json:"plan"`
	ExternalID  string `json:"external_id"`
}

// OAuthState holds oauth validation state
type OAuthState struct {
	State     string    `json:"state"`
	Timestamp time.Time `json:"ts"`
}
