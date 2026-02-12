package db

import (
	"context"

	"github.com/numbergroup/claw-swarm/pkg/types"
)

type UserDB interface {
	GetByID(ctx context.Context, id string) (types.User, error)
	GetByEmail(ctx context.Context, email string) (types.User, error)
	Insert(ctx context.Context, user types.User) (string, error)
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
}

type BotSpaceDB interface {
	GetByID(ctx context.Context, id string) (types.BotSpace, error)
	ListByUserID(ctx context.Context, userID string) ([]types.BotSpace, error)
	GetByJoinCode(ctx context.Context, joinCode string) (types.BotSpace, error)
	Insert(ctx context.Context, botSpace types.BotSpace) (string, error)
	Update(ctx context.Context, id string, name string, description *string) (types.BotSpace, error)
	Delete(ctx context.Context, id string) error
	UpdateJoinCodes(ctx context.Context, id string, joinCode string, managerJoinCode string) (types.BotSpace, error)
	SetManagerBotID(ctx context.Context, id string, botID string) error
	ClearManagerBotID(ctx context.Context, id string) error
}

type SpaceMemberDB interface {
	Insert(ctx context.Context, member types.SpaceMember) (string, error)
	ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.SpaceMemberWithUser, error)
	Delete(ctx context.Context, botSpaceID string, userID string) error
	IsMember(ctx context.Context, botSpaceID string, userID string) (bool, error)
}

type BotDB interface {
	GetByID(ctx context.Context, id string) (types.Bot, error)
	ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.Bot, error)
	Insert(ctx context.Context, bot types.Bot) (string, error)
	Delete(ctx context.Context, id string) error
	SetManager(ctx context.Context, id string, isManager bool) error
	UpdateLastSeen(ctx context.Context, id string) error
}

type MessageDB interface {
	Insert(ctx context.Context, msg types.Message) (string, error)
	ListByBotSpaceID(ctx context.Context, botSpaceID string, limit int, before *string) ([]types.Message, error)
	ListSince(ctx context.Context, botSpaceID string, sinceID string, limit int) ([]types.Message, error)
	ListSpaceIDsExceedingCount(ctx context.Context, maxCount int) ([]string, error)
	DeleteOlderThanNth(ctx context.Context, botSpaceID string, keep int) (int64, error)
}

type BotStatusDB interface {
	GetByBotSpaceIDAndBotID(ctx context.Context, botSpaceID string, botID string) (types.BotStatus, error)
	ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.BotStatus, error)
	Upsert(ctx context.Context, status types.BotStatus) (types.BotStatus, error)
	BulkUpsert(ctx context.Context, statuses []types.BotStatus) ([]types.BotStatus, error)
}

type SummaryDB interface {
	GetByBotSpaceID(ctx context.Context, botSpaceID string) (types.Summary, error)
	Upsert(ctx context.Context, summary types.Summary) (types.Summary, error)
}

type InviteCodeDB interface {
	Insert(ctx context.Context, code types.InviteCode) (string, error)
	GetByCode(ctx context.Context, code string) (types.InviteCode, error)
	ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.InviteCode, error)
	Delete(ctx context.Context, id string) error
}

type SpaceTaskDB interface {
	Insert(ctx context.Context, task types.SpaceTask) (types.SpaceTask, error)
	GetByID(ctx context.Context, id string) (types.SpaceTask, error)
	ListByBotSpaceID(ctx context.Context, botSpaceID string, status *string) ([]types.SpaceTask, error)
	GetActiveByBotID(ctx context.Context, botSpaceID string, botID string) (*types.SpaceTask, error)
	Update(ctx context.Context, task types.SpaceTask) (types.SpaceTask, error)
}

type BotSkillDB interface {
	Insert(ctx context.Context, skill types.BotSkill) (types.BotSkill, error)
	GetByID(ctx context.Context, id string) (types.BotSkill, error)
	ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.BotSkill, error)
	Update(ctx context.Context, skill types.BotSkill) (types.BotSkill, error)
	Delete(ctx context.Context, id string) error
}
