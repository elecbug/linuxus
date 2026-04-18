package packet

import "time"

// UserUpRequest is the request payload for runtime preparation.
type UserUpRequest struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// SafeID is the sanitized identifier used for runtime resources.
	SafeID string `json:"safe_id"`
}

// UserUpResponse is the result payload for runtime preparation.
type UserUpResponse struct {
	// OK indicates whether runtime preparation succeeded.
	OK bool `json:"ok"`
	// UserID echoes the requested user identifier.
	UserID string `json:"user_id,omitempty"`
	// SafeID echoes the sanitized identifier.
	SafeID string `json:"safe_id,omitempty"`
	// ContainerName is the prepared runtime container name.
	ContainerName string `json:"container_name,omitempty"`
	// NetworkName is the prepared runtime network name.
	NetworkName string `json:"network_name,omitempty"`
	// Subnet is the allocated runtime subnet.
	Subnet string `json:"subnet,omitempty"`
	// Message contains human-readable status details.
	Message string `json:"message"`
}

// SessionStateReport defines the structure of session state reports sent to the manager.
type SessionStateReport struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// ActiveSessions is the count of active sessions for the user.
	ActiveSessions int `json:"active_sessions"`
	// ObservedAt is the timestamp when the session state was observed.
	ObservedAt time.Time `json:"observed_at"`
}
