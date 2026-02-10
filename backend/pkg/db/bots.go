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

type botDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	getByID          *sqlx.Stmt
	listByBotSpaceID *sqlx.Stmt
	insert           *sqlx.NamedStmt
	deleteStmt       *sqlx.Stmt
	setManager       *sqlx.Stmt
	updateLastSeen   *sqlx.Stmt
}

func NewBotDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (BotDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.Bot]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.Bot]()

	getByID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bots WHERE id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByID statement")
	}

	listByBotSpaceID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bots WHERE bot_space_id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByBotSpaceID statement")
	}

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO bots (%s) VALUES (:%s) RETURNING id`,
		colStr, strings.Join(rawCols, ", :")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	deleteStmt, err := sdb.PreparexContext(ctx,
		`DELETE FROM bots WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare delete statement")
	}

	setManager, err := sdb.PreparexContext(ctx,
		`UPDATE bots SET is_manager = $1, updated_at = now() WHERE id = $2`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare setManager statement")
	}

	updateLastSeen, err := sdb.PreparexContext(ctx,
		`UPDATE bots SET last_seen_at = now() WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare updateLastSeen statement")
	}

	return &botDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		getByID:          getByID,
		listByBotSpaceID: listByBotSpaceID,
		insert:           insert,
		deleteStmt:       deleteStmt,
		setManager:       setManager,
		updateLastSeen:   updateLastSeen,
	}, nil
}

func (b *botDB) GetByID(ctx context.Context, id string) (types.Bot, error) {
	var bot types.Bot
	err := b.getByID.GetContext(ctx, &bot, id)
	if err != nil {
		return bot, errors.Wrap(err, "failed to get bot by id")
	}
	return bot, nil
}

func (b *botDB) ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.Bot, error) {
	bots := make([]types.Bot, 0)
	err := b.listByBotSpaceID.SelectContext(ctx, &bots, botSpaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list bots by bot space id")
	}
	return bots, nil
}

func (b *botDB) Insert(ctx context.Context, bot types.Bot) (string, error) {
	var id string
	err := b.insert.GetContext(ctx, &id, bot)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert bot")
	}
	return id, nil
}

func (b *botDB) Delete(ctx context.Context, id string) error {
	_, err := b.deleteStmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete bot")
	}
	return nil
}

func (b *botDB) SetManager(ctx context.Context, id string, isManager bool) error {
	_, err := b.setManager.ExecContext(ctx, isManager, id)
	if err != nil {
		return errors.Wrap(err, "failed to set manager")
	}
	return nil
}

func (b *botDB) UpdateLastSeen(ctx context.Context, id string) error {
	_, err := b.updateLastSeen.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to update last seen")
	}
	return nil
}
