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

type messageDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	insert           *sqlx.NamedStmt
	listRecent       *sqlx.Stmt
	listBeforeCursor *sqlx.Stmt
	listSinceCursor  *sqlx.Stmt
	getCreatedAt     *sqlx.Stmt
}

func NewMessageDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (MessageDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.Message]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.Message]()

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO messages (%s) VALUES (:%s) RETURNING id`,
		colStr, strings.Join(rawCols, ", :")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	listRecent, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM messages
		WHERE bot_space_id = $1
		ORDER BY created_at DESC LIMIT $2`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listRecent statement")
	}

	listBeforeCursor, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM messages
		WHERE bot_space_id = $1 AND created_at < $2
		ORDER BY created_at DESC LIMIT $3`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listBeforeCursor statement")
	}

	listSinceCursor, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM messages
		WHERE bot_space_id = $1 AND created_at > $2
		ORDER BY created_at ASC LIMIT $3`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listSinceCursor statement")
	}

	getCreatedAt, err := sdb.PreparexContext(ctx,
		`SELECT created_at FROM messages WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getCreatedAt statement")
	}

	return &messageDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		insert:           insert,
		listRecent:       listRecent,
		listBeforeCursor: listBeforeCursor,
		listSinceCursor:  listSinceCursor,
		getCreatedAt:     getCreatedAt,
	}, nil
}

func (m *messageDB) Insert(ctx context.Context, msg types.Message) (string, error) {
	var id string
	err := m.insert.GetContext(ctx, &id, msg)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert message")
	}
	return id, nil
}

func (m *messageDB) ListByBotSpaceID(ctx context.Context, botSpaceID string, limit int, before *string) ([]types.Message, error) {
	var messages []types.Message

	if before == nil {
		err := m.listRecent.SelectContext(ctx, &messages, botSpaceID, limit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list recent messages")
		}
		return messages, nil
	}

	var cursorTime any
	err := m.getCreatedAt.GetContext(ctx, &cursorTime, *before)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cursor message created_at")
	}

	err = m.listBeforeCursor.SelectContext(ctx, &messages, botSpaceID, cursorTime, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list messages before cursor")
	}
	return messages, nil
}

func (m *messageDB) ListSince(ctx context.Context, botSpaceID string, sinceID string, limit int) ([]types.Message, error) {
	var cursorTime any
	err := m.getCreatedAt.GetContext(ctx, &cursorTime, sinceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cursor message created_at")
	}

	var messages []types.Message
	err = m.listSinceCursor.SelectContext(ctx, &messages, botSpaceID, cursorTime, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list messages since cursor")
	}
	return messages, nil
}
