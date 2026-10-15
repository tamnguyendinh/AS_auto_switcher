package connector

// Connect-RPC & HTTP Endpoint definitions for Antigravity LanguageServerService
const (
	EndpointGetUserStatus          = "/exa.language_server_pb.LanguageServerService/GetUserStatus"
	EndpointAuthLogout             = "/exa.language_server_pb.LanguageServerService/AuthLogout"
	EndpointReauth                 = "/exa.language_server_pb.LanguageServerService/ReAuth"
	EndpointReload                 = "/exa.language_server_pb.LanguageServerService/Reload"
	EndpointLoadTrajectory         = "/exa.language_server_pb.LanguageServerService/LoadTrajectory"
	EndpointSmartFocusConversation = "/exa.language_server_pb.LanguageServerService/SmartFocusConversation"
	EndpointCascadeModelConfigData = "/exa.language_server_pb.LanguageServerService/GetCascadeModelConfigData"
	EndpointSendUserCascadeMessage = "/exa.language_server_pb.LanguageServerService/SendUserCascadeMessage"

	DefaultActiveAgentModel = "MODEL_PLACEHOLDER_M318"
)

// Candidate endpoints for fallback resilience across language server versions.
var (
	DefaultUserStatusEndpoints = []string{
		EndpointGetUserStatus,
		"/GetUserStatus",
		"/user_status",
	}

	DefaultAuthLogoutEndpoints = []string{
		EndpointAuthLogout,
		"/AuthLogout",
		"/logout",
	}

	DefaultReauthEndpoints = []string{
		EndpointReauth,
		EndpointReload,
		"/ReAuth",
		"/reauth",
		"/reload",
	}
)

// UserStatus holds the verified active user identity currently loaded in language_server RAM.
type UserStatus struct {
	Email       string                 `json:"email"`
	Name        string                 `json:"name,omitempty"`
	DisplayName string                 `json:"display_name,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Raw         map[string]interface{} `json:"raw,omitempty"`
}

// CascadeMessageItem represents a single message segment in a Cascade interaction.
type CascadeMessageItem struct {
	Text string `json:"text"`
}

// PlannerConfig represents planner model configurations for Cascade queries.
type PlannerConfig struct {
	PlanModel      string            `json:"planModel"`
	RequestedModel map[string]string `json:"requestedModel"`
}

// CascadeConfig represents the inner configuration envelope for Cascade prompt execution.
type CascadeConfig struct {
	PlannerConfig PlannerConfig `json:"plannerConfig"`
}

// SendCascadeMessagePayload encapsulates the request body for SendUserCascadeMessage.
type SendCascadeMessagePayload struct {
	CascadeID     string               `json:"cascadeId"`
	Items         []CascadeMessageItem `json:"items"`
	CascadeConfig interface{}          `json:"cascadeConfig,omitempty"`
}

// LoadTrajectoryPayload encapsulates the request body for LoadTrajectory / SmartFocus.
type TrajectoryPayload struct {
	CascadeID string `json:"cascadeId"`
}
