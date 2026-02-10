package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/innodv/psql"
	"github.com/jmoiron/sqlx"
	"github.com/numbergroup/claw-swarm/pkg/config"
	"github.com/numbergroup/claw-swarm/pkg/types"
	"github.com/numbergroup/errors"
	"github.com/sirupsen/logrus"
)

type spaceMemberDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	insert           *sqlx.NamedStmt
	listByBotSpaceID *sqlx.Stmt
	deleteStmt       *sqlx.Stmt
	isMember         *sqlx.Stmt
}

func NewSpaceMemberDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (SpaceMemberDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.SpaceMember]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.SpaceMember]()

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO space_members (%s) VALUES (:%s) RETURNING id`,
		colStr, strings.Join(rawCols, ", :")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	listByBotSpaceID, err := sdb.PreparexContext(ctx,
		`SELECT sm.id, sm.bot_space_id, sm.user_id, sm.role, sm.joined_at,
		        u.email, u.display_name
		FROM space_members sm
		INNER JOIN users u ON u.id = sm.user_id
		WHERE sm.bot_space_id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByBotSpaceID statement")
	}

	deleteStmt, err := sdb.PreparexContext(ctx,
		`DELETE FROM space_members WHERE bot_space_id = $1 AND user_id = $2`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare delete statement")
	}

	isMemberStmt, err := sdb.PreparexContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM space_members WHERE bot_space_id = $1 AND user_id = $2)`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare isMember statement")
	}

	return &spaceMemberDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		insert:           insert,
		listByBotSpaceID: listByBotSpaceID,
		deleteStmt:       deleteStmt,
		isMember:         isMemberStmt,
	}, nil
}

func (s *spaceMemberDB) Insert(ctx context.Context, member types.SpaceMember) (string, error) {
	var id string
	err := s.insert.GetContext(ctx, &id, member)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert space member")
	}
	return id, nil
}

func (s *spaceMemberDB) ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.SpaceMemberWithUser, error) {
	members := make([]types.SpaceMemberWithUser, 0)
	err := s.listByBotSpaceID.SelectContext(ctx, &members, botSpaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list space members")
	}
	return members, nil
}

func (s *spaceMemberDB) Delete(ctx context.Context, botSpaceID string, userID string) error {
	_, err := s.deleteStmt.ExecContext(ctx, botSpaceID, userID)
	if err != nil {
		return errors.Wrap(err, "failed to delete space member")
	}
	return nil
}

func (s *spaceMemberDB) IsMember(ctx context.Context, botSpaceID string, userID string) (bool, error) {
	var exists bool
	err := s.isMember.GetContext(ctx, &exists, botSpaceID, userID)
	if err != nil {
		return false, errors.Wrap(err, "failed to check membership")
	}
	return exists, nil
}
