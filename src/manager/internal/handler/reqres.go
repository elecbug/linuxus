package handler

// UserUpRequest is the request payload for runtime preparation.
type UserUpRequest struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// SafeID is the sanitized identifier used for naming resources.
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
	// NetworkName is the attached runtime network name.
	NetworkName string `json:"network_name,omitempty"`
	// Subnet is the allocated network subnet.
	Subnet string `json:"subnet,omitempty"`
	// Message contains human-readable status details.
	Message string `json:"message"`
}
