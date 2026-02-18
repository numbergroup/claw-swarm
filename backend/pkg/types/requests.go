package types

type SignupRequest struct {
	Email string `json:"email" binding:"required,email"`
	// #nosec G117
	Password    string  `json:"password" binding:"required,min=8"`
	DisplayName *string `json:"displayName"`
}

type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
	// #nosec G117
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type BotRegistrationRequest struct {
	JoinCode     string `json:"joinCode" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Capabilities string `json:"capabilities" binding:"required"`
}

type BotSpaceBasic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type BotRegistrationResponse struct {
	Token    string        `json:"token"`
	Bot      Bot           `json:"bot"`
	BotSpace BotSpaceBasic `json:"botSpace"`
}

type CreateBotSpaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

type UpdateBotSpaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type PostMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type MessageListResponse struct {
	Messages []Message `json:"messages"`
	Count    int       `json:"count"`
	HasMore  bool      `json:"hasMore"`
}

type UpdateBotStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type BulkStatusItem struct {
	BotID  string `json:"botId" binding:"required"`
	Status string `json:"status" binding:"required"`
}

type BulkUpdateBotStatusRequest struct {
	Statuses []BulkStatusItem `json:"statuses" binding:"required,dive"`
}

type UpdateSummaryRequest struct {
	Content string `json:"content" binding:"required"`
}

type OverallResponse struct {
	Messages MessageListResponse `json:"messages"`
	Summary  *Summary            `json:"summary"`
}

type JoinBotSpaceRequest struct {
	InviteCode string `json:"inviteCode" binding:"required"`
}

type CreateSpaceTaskRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	BotID       *string `json:"botId"`
}

type AssignTaskRequest struct {
	BotID string `json:"botId" binding:"required"`
}

type ArtifactListResponse struct {
	Artifacts []Artifact `json:"artifacts"`
	Count     int        `json:"count"`
	HasMore   bool       `json:"hasMore"`
}

type CreateArtifactRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Data        string `json:"data" binding:"required"`
}

type CreateBotSkillRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Tags        []string `json:"tags"`
}

type UpdateBotSkillRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
}
