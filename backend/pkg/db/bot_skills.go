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

type botSkillDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	insert           *sqlx.NamedStmt
	getByID          *sqlx.Stmt
	listByBotSpaceID *sqlx.Stmt
	update           *sqlx.NamedStmt
	deleteStmt       *sqlx.Stmt
}

func NewBotSkillDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (BotSkillDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.BotSkill]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.BotSkill]()

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO bot_skills (%s) VALUES (:%s) RETURNING %s`,
		colStr, strings.Join(rawCols, ", :"), colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	getByID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bot_skills WHERE id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByID statement")
	}

	listByBotSpaceID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM bot_skills WHERE bot_space_id = $1 ORDER BY created_at ASC`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByBotSpaceID statement")
	}

	update, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`UPDATE bot_skills SET name = :name, description = :description, tags = :tags,
		bot_name = :bot_name, updated_at = :updated_at WHERE id = :id RETURNING %s`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare update statement")
	}

	deleteStmt, err := sdb.PreparexContext(ctx, `DELETE FROM bot_skills WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare delete statement")
	}

	return &botSkillDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		insert:           insert,
		getByID:          getByID,
		listByBotSpaceID: listByBotSpaceID,
		update:           update,
		deleteStmt:       deleteStmt,
	}, nil
}

func (b *botSkillDB) Insert(ctx context.Context, skill types.BotSkill) (types.BotSkill, error) {
	var result types.BotSkill
	err := b.insert.GetContext(ctx, &result, skill)
	if err != nil {
		return result, errors.Wrap(err, "failed to insert bot skill")
	}
	return result, nil
}

func (b *botSkillDB) GetByID(ctx context.Context, id string) (types.BotSkill, error) {
	var skill types.BotSkill
	err := b.getByID.GetContext(ctx, &skill, id)
	if err != nil {
		return skill, errors.Wrap(err, "failed to get bot skill")
	}
	return skill, nil
}

func (b *botSkillDB) ListByBotSpaceID(ctx context.Context, botSpaceID string) ([]types.BotSkill, error) {
	skills := make([]types.BotSkill, 0)
	err := b.listByBotSpaceID.SelectContext(ctx, &skills, botSpaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list bot skills")
	}
	return skills, nil
}

func (b *botSkillDB) Update(ctx context.Context, skill types.BotSkill) (types.BotSkill, error) {
	var result types.BotSkill
	err := b.update.GetContext(ctx, &result, skill)
	if err != nil {
		return result, errors.Wrap(err, "failed to update bot skill")
	}
	return result, nil
}

func (b *botSkillDB) Delete(ctx context.Context, id string) error {
	_, err := b.deleteStmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete bot skill")
	}
	return nil
}
