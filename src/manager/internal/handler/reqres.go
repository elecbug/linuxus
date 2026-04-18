package handler

import "time"

// userUpRequest is the request payload for runtime preparation.
type userUpRequest struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	SafeID string `json:"safe_id"`
}

// userUpResponse is the result payload for runtime preparation.
type userUpResponse struct {
	// OK indicates whether runtime preparation succeeded.
	OK bool `json:"ok"`
	// UserID echoes the requested user identifier.
	UserID string `json:"user_id,omitempty"`
	// SafeID echoes the sanitized identifier.
	SafeID string `json:"safe_id,omitempty"`
	// ContainerName is the prepared runtime container name.
	ContainerName string `json:"container_name,omitempty"`
	NetworkName   string `json:"network_name,omitempty"`
	Subnet        string `json:"subnet,omitempty"`
	Message       string `json:"message"`
}

// sessionStateReport defines the structure of session state reports sent to the manager.
type sessionStateReport struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// ActiveSessions is the count of active sessions for the user.
	ActiveSessions int `json:"active_sessions"`
	// ObservedAt is the timestamp when the session state was observed.
	ObservedAt time.Time `json:"observed_at"`
}
