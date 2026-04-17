package handler

type UserUpRequest struct {
	UserID string `json:"user_id"`
	SafeID string `json:"safe_id"`
}

type UserUpResponse struct {
	OK            bool   `json:"ok"`
	UserID        string `json:"user_id,omitempty"`
	SafeID        string `json:"safe_id,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
	NetworkName   string `json:"network_name,omitempty"`
	Subnet        string `json:"subnet,omitempty"`
	Message       string `json:"message"`
}
