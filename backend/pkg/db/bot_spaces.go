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

type botSpaceDB struct {
	db                *sqlx.DB
	log               logrus.Ext1FieldLogger
	conf              *config.Config
	getByID           *sqlx.Stmt
	listByUserID      *sqlx.Stmt
	getByJoinCode     *sqlx.Stmt
	insert            *sqlx.NamedStmt
	update            *sqlx.Stmt
	deleteStmt        *sqlx.Stmt
	updateJoinCodes   *sqlx.Stmt
	setManagerBotID   *sqlx.Stmt
	clearManagerBotID *sqlx.Stmt
}

func NewBotSpaceDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (BotSpaceDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.BotSpace]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.BotSpace]()

	prefixedCols := make([]string, len(cols))
	for i, c := range cols {
		prefixedCols[i] = "bs." + c
	}
	prefixedColStr := strings.Join(prefixedCols, ", ")

	getByID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bot_spaces WHERE id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByID statement")
	}

	listByUserID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bot_spaces bs
		INNER JOIN space_members sm ON sm.bot_space_id = bs.id
		WHERE sm.user_id = $1
		ORDER BY bs.created_at DESC`, prefixedColStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByUserID statement")
	}

	getByJoinCode, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bot_spaces WHERE join_code = $1 OR manager_join_code = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByJoinCode statement")
	}

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO bot_spaces (%s) VALUES (:%s) RETURNING id`,
		colStr, strings.Join(rawCols, ", :")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	update, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`UPDATE bot_spaces SET name = $1, description = $2, updated_at = now()
		WHERE id = $3 RETURNING %s`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare update statement")
	}

	deleteStmt, err := sdb.PreparexContext(ctx,
		`DELETE FROM bot_spaces WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare delete statement")
	}

	updateJoinCodes, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`UPDATE bot_spaces SET join_code = $1, manager_join_code = $2, updated_at = now()
		WHERE id = $3 RETURNING %s`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare updateJoinCodes statement")
	}

	setManagerBotID, err := sdb.PreparexContext(ctx,
		`UPDATE bot_spaces SET manager_bot_id = $1, updated_at = now() WHERE id = $2`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare setManagerBotID statement")
	}

	clearManagerBotID, err := sdb.PreparexContext(ctx,
		`UPDATE bot_spaces SET manager_bot_id = NULL, updated_at = now() WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare clearManagerBotID statement")
	}

	return &botSpaceDB{
		db:                sdb,
		log:               conf.GetLogger(),
		conf:              conf,
		getByID:           getByID,
		listByUserID:      listByUserID,
		getByJoinCode:     getByJoinCode,
		insert:            insert,
		update:            update,
		deleteStmt:        deleteStmt,
		updateJoinCodes:   updateJoinCodes,
		setManagerBotID:   setManagerBotID,
		clearManagerBotID: clearManagerBotID,
	}, nil
}

func (b *botSpaceDB) GetByID(ctx context.Context, id string) (types.BotSpace, error) {
	var bs types.BotSpace
	err := b.getByID.GetContext(ctx, &bs, id)
	if err != nil {
		return bs, errors.Wrap(err, "failed to get bot space by id")
	}
	return bs, nil
}

func (b *botSpaceDB) ListByUserID(ctx context.Context, userID string) ([]types.BotSpace, error) {
	spaces := make([]types.BotSpace, 0)
	err := b.listByUserID.SelectContext(ctx, &spaces, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list bot spaces by user id")
	}
	return spaces, nil
}

func (b *botSpaceDB) GetByJoinCode(ctx context.Context, joinCode string) (types.BotSpace, error) {
	var bs types.BotSpace
	err := b.getByJoinCode.GetContext(ctx, &bs, joinCode)
	if err != nil {
		return bs, errors.Wrap(err, "failed to get bot space by join code")
	}
	return bs, nil
}

func (b *botSpaceDB) Insert(ctx context.Context, botSpace types.BotSpace) (string, error) {
	var id string
	err := b.insert.GetContext(ctx, &id, botSpace)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert bot space")
	}
	return id, nil
}

func (b *botSpaceDB) Update(ctx context.Context, id string, name string, description *string) (types.BotSpace, error) {
	var bs types.BotSpace
	err := b.update.GetContext(ctx, &bs, name, description, id)
	if err != nil {
		return bs, errors.Wrap(err, "failed to update bot space")
	}
	return bs, nil
}

func (b *botSpaceDB) Delete(ctx context.Context, id string) error {
	_, err := b.deleteStmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete bot space")
	}
	return nil
}

func (b *botSpaceDB) UpdateJoinCodes(ctx context.Context, id string, joinCode string, managerJoinCode string) (types.BotSpace, error) {
	var bs types.BotSpace
	err := b.updateJoinCodes.GetContext(ctx, &bs, joinCode, managerJoinCode, id)
	if err != nil {
		return bs, errors.Wrap(err, "failed to update join codes")
	}
	return bs, nil
}

func (b *botSpaceDB) SetManagerBotID(ctx context.Context, id string, botID string) error {
	_, err := b.setManagerBotID.ExecContext(ctx, botID, id)
	if err != nil {
		return errors.Wrap(err, "failed to set manager bot id")
	}
	return nil
}

func (b *botSpaceDB) ClearManagerBotID(ctx context.Context, id string) error {
	_, err := b.clearManagerBotID.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to clear manager bot id")
	}
	return nil
}
