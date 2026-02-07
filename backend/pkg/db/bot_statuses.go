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

type botStatusDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	getByPair        *sqlx.Stmt
	listByBotSpaceID *sqlx.Stmt
	upsert           *sqlx.NamedStmt
}

func NewBotStatusDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (BotStatusDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.BotStatus]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.BotStatus]()

	getByPair, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bot_statuses WHERE bot_space_id = $1 AND bot_id = $2`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByPair statement")
	}

	listByBotSpaceID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bot_statuses WHERE bot_space_id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByBotSpaceID statement")
	}

	upsert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO bot_statuses (%s) VALUES (:%s)
		ON CONFLICT (bot_space_id, bot_id)
		DO UPDATE SET status = EXCLUDED.status, bot_name = EXCLUDED.bot_name,
		             updated_by_bot_id = EXCLUDED.updated_by_bot_id, updated_at = now()
		RETURNING %s`,
		colStr, strings.Join(rawCols, ", :"), colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare upsert statement")
	}

	return &botStatusDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		getByPair:        getByPair,
		listByBotSpaceID: listByBotSpaceID,
		upsert:           upsert,
	}, nil
}

func (b *botStatusDB) GetByBotSpaceIDAndBotID(ctx context.Context, botSpaceID string, botID string) (types.BotStatus, error) {
	var status types.BotStatus
	err := b.getByPair.GetContext(ctx, &status, botSpaceID, botID)
	if err != nil {
		return status, errors.Wrap(err, "failed to get bot status")
	}
	return status, nil
}

func (b *botStatusDB) ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.BotStatus, error) {
	var statuses []types.BotStatus
	err := b.listByBotSpaceID.SelectContext(ctx, &statuses, botSpaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list bot statuses")
	}
	return statuses, nil
}

func (b *botStatusDB) Upsert(ctx context.Context, status types.BotStatus) (types.BotStatus, error) {
	var result types.BotStatus
	err := b.upsert.GetContext(ctx, &result, status)
	if err != nil {
		return result, errors.Wrap(err, "failed to upsert bot status")
	}
	return result, nil
}

func (b *botStatusDB) BulkUpsert(ctx context.Context, statuses []types.BotStatus) ([]types.BotStatus, error) {
	tx, err := b.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	txUpsert := tx.NamedStmt(b.upsert)
	results := make([]types.BotStatus, 0, len(statuses))
	for _, s := range statuses {
		var result types.BotStatus
		err := txUpsert.GetContext(ctx, &result, s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to upsert bot status in bulk")
		}
		results = append(results, result)
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to commit bulk upsert transaction")
	}
	return results, nil
}
