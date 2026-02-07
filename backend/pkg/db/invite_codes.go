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

type inviteCodeDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	insert           *sqlx.NamedStmt
	getByCode        *sqlx.Stmt
	listByBotSpaceID *sqlx.Stmt
	deleteStmt       *sqlx.Stmt
}

func NewInviteCodeDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (InviteCodeDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.InviteCode]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.InviteCode]()

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO invite_codes (%s) VALUES (:%s) RETURNING id`,
		colStr, strings.Join(rawCols, ", :")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	getByCode, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM invite_codes WHERE code = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByCode statement")
	}

	listByBotSpaceID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM invite_codes WHERE bot_space_id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByBotSpaceID statement")
	}

	deleteStmt, err := sdb.PreparexContext(ctx,
		`DELETE FROM invite_codes WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare delete statement")
	}

	return &inviteCodeDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		insert:           insert,
		getByCode:        getByCode,
		listByBotSpaceID: listByBotSpaceID,
		deleteStmt:       deleteStmt,
	}, nil
}

func (ic *inviteCodeDB) Insert(ctx context.Context, code types.InviteCode) (string, error) {
	var id string
	err := ic.insert.GetContext(ctx, &id, code)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert invite code")
	}
	return id, nil
}

func (ic *inviteCodeDB) GetByCode(ctx context.Context, code string) (types.InviteCode, error) {
	var inviteCode types.InviteCode
	err := ic.getByCode.GetContext(ctx, &inviteCode, code)
	if err != nil {
		return inviteCode, errors.Wrap(err, "failed to get invite code")
	}
	return inviteCode, nil
}

func (ic *inviteCodeDB) ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.InviteCode, error) {
	var codes []types.InviteCode
	err := ic.listByBotSpaceID.SelectContext(ctx, &codes, botSpaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list invite codes")
	}
	return codes, nil
}

func (ic *inviteCodeDB) Delete(ctx context.Context, id string) error {
	_, err := ic.deleteStmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete invite code")
	}
	return nil
}
