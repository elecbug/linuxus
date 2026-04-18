package packet

import (
	"time"
)

type UserUpRequest struct {
	UserID string `json:"user_id"`
	SafeID string `json:"safe_id"`
}

type UserUpResponse struct {
	OK            bool   `json:"ok"`
	UserID        string `json:"user_id"`
	SafeID        string `json:"safe_id"`
	ContainerName string `json:"container_name"`
	NetworkName   string `json:"network_name"`
	Subnet        string `json:"subnet"`
	Message       string `json:"message"`
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
