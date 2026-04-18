package packet

import (
	"time"
)

// UserUpRequest requests startup/validation of a user runtime from manager.
type UserUpRequest struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// SafeID is the sanitized identifier used for runtime resources.
	SafeID string `json:"safe_id"`
}

// UserUpResponse is the manager response for a user runtime preparation request.
type UserUpResponse struct {
	// OK indicates whether the runtime is ready.
	OK bool `json:"ok"`
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// SafeID is the sanitized identifier used for runtime resources.
	SafeID string `json:"safe_id"`
	// ContainerName is the prepared runtime container name.
	ContainerName string `json:"container_name"`
	// NetworkName is the prepared runtime network name.
	NetworkName string `json:"network_name"`
	// Subnet is the allocated runtime subnet.
	Subnet string `json:"subnet"`
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
