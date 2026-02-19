package types

import (
	"time"

	"github.com/lib/pq"
	pgvector "github.com/pgvector/pgvector-go"
)

type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	DisplayName  *string   `json:"displayName" db:"display_name"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type BotSpace struct {
	ID              string    `json:"id" db:"id"`
	OwnerID         string    `json:"ownerId" db:"owner_id"`
	Name            string    `json:"name" db:"name"`
	Description     *string   `json:"description" db:"description"`
	JoinCode        string    `json:"joinCode" db:"join_code"`
	ManagerJoinCode string    `json:"managerJoinCode" db:"manager_join_code"`
	ManagerBotID    *string   `json:"managerBotId" db:"manager_bot_id"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

type SpaceMember struct {
	ID         string    `json:"id" db:"id"`
	BotSpaceID string    `json:"botSpaceId" db:"bot_space_id"`
	UserID     string    `json:"userId" db:"user_id"`
	Role       string    `json:"role" db:"role"`
	JoinedAt   time.Time `json:"joinedAt" db:"joined_at"`
}

type SpaceMemberWithUser struct {
	SpaceMember
	Email       string  `json:"email" db:"email"`
	DisplayName *string `json:"displayName" db:"display_name"`
}

type Bot struct {
	ID           string     `json:"id" db:"id"`
	BotSpaceID   string     `json:"botSpaceId" db:"bot_space_id"`
	Name         string     `json:"name" db:"name"`
	Capabilities *string    `json:"capabilities" db:"capabilities"`
	IsManager    bool       `json:"isManager" db:"is_manager"`
	IsMuted      bool       `json:"isMuted" db:"is_muted"`
	LastSeenAt   *time.Time `json:"lastSeenAt" db:"last_seen_at"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" db:"updated_at"`
}

type Message struct {
	ID         string    `json:"id" db:"id"`
	BotSpaceID string    `json:"botSpaceId" db:"bot_space_id"`
	SenderID   string    `json:"senderId" db:"sender_id"`
	SenderName string    `json:"senderName" db:"sender_name"`
	SenderType string    `json:"senderType" db:"sender_type"`
	Content    string    `json:"content" db:"content"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
}

type BotStatus struct {
	ID             string    `json:"id" db:"id"`
	BotSpaceID     string    `json:"botSpaceId" db:"bot_space_id"`
	BotID          string    `json:"botId" db:"bot_id"`
	BotName        string    `json:"botName" db:"bot_name"`
	Status         string    `json:"status" db:"status"`
	UpdatedByBotID string    `json:"updatedByBotId" db:"updated_by_bot_id"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type Summary struct {
	ID             string    `json:"id" db:"id"`
	BotSpaceID     string    `json:"botSpaceId" db:"bot_space_id"`
	Content        string    `json:"content" db:"content"`
	CreatedByBotID string    `json:"createdByBotId" db:"created_by_bot_id"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type InviteCode struct {
	ID         string     `json:"id" db:"id"`
	BotSpaceID string     `json:"botSpaceId" db:"bot_space_id"`
	Code       string     `json:"code" db:"code"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	ExpiresAt  *time.Time `json:"expiresAt" db:"expires_at"`
}

type SpaceTask struct {
	ID             string     `json:"id" db:"id"`
	BotSpaceID     string     `json:"botSpaceId" db:"bot_space_id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description" db:"description"`
	Status         string     `json:"status" db:"status"`
	BotID          *string    `json:"botId" db:"bot_id"`
	CreatedByBotID string     `json:"createdByBotId" db:"created_by_bot_id"`
	CompletedAt    *time.Time `json:"completedAt" db:"completed_at"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`
}

type Artifact struct {
	ID             string    `json:"id" db:"id"`
	BotSpaceID     string    `json:"botSpaceId" db:"bot_space_id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	Data           string    `json:"data" db:"data"`
	CreatedByBotID string    `json:"createdByBotId" db:"created_by_bot_id"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type BotSkill struct {
	ID          string           `json:"id" db:"id"`
	BotSpaceID  string           `json:"botSpaceId" db:"bot_space_id"`
	BotID       string           `json:"botId" db:"bot_id"`
	BotName     string           `json:"botName" db:"bot_name"`
	Name        string           `json:"name" db:"name"`
	Description string           `json:"description" db:"description"`
	Tags        pq.StringArray   `json:"tags" db:"tags"`
	Embedding   *pgvector.Vector `json:"embedding" db:"embedding"`
	CreatedAt   time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time        `json:"updatedAt" db:"updated_at"`
}
