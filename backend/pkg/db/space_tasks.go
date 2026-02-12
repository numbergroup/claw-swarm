package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/innodv/psql"
	"github.com/jmoiron/sqlx"
	"github.com/numbergroup/claw-swarm/pkg/config"
	"github.com/numbergroup/claw-swarm/pkg/types"
	"github.com/numbergroup/errors"
	"github.com/sirupsen/logrus"
)

type spaceTaskDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	insert           *sqlx.NamedStmt
	getByID          *sqlx.Stmt
	listByBotSpaceID *sqlx.Stmt
	listByStatus     *sqlx.Stmt
	getActiveByBotID *sqlx.Stmt
	update           *sqlx.NamedStmt
}

func NewSpaceTaskDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (SpaceTaskDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.SpaceTask]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.SpaceTask]()

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO space_tasks (%s) VALUES (:%s) RETURNING %s`,
		colStr, strings.Join(rawCols, ", :"), colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	getByID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM space_tasks WHERE id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByID statement")
	}

	listByBotSpaceID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM space_tasks WHERE bot_space_id = $1 ORDER BY created_at ASC`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByBotSpaceID statement")
	}

	listByStatus, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM space_tasks WHERE bot_space_id = $1 AND status = $2 ORDER BY created_at ASC`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listByStatus statement")
	}

	getActiveByBotID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM space_tasks WHERE bot_space_id = $1 AND bot_id = $2 AND status = 'in_progress' LIMIT 1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getActiveByBotID statement")
	}

	update, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`UPDATE space_tasks SET status = :status, bot_id = :bot_id, completed_at = :completed_at,
		updated_at = :updated_at WHERE id = :id RETURNING %s`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare update statement")
	}

	return &spaceTaskDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		insert:           insert,
		getByID:          getByID,
		listByBotSpaceID: listByBotSpaceID,
		listByStatus:     listByStatus,
		getActiveByBotID: getActiveByBotID,
		update:           update,
	}, nil
}

func (s *spaceTaskDB) Insert(ctx context.Context, task types.SpaceTask) (types.SpaceTask, error) {
	var result types.SpaceTask
	err := s.insert.GetContext(ctx, &result, task)
	if err != nil {
		return result, errors.Wrap(err, "failed to insert space task")
	}
	return result, nil
}

func (s *spaceTaskDB) GetByID(ctx context.Context, id string) (types.SpaceTask, error) {
	var task types.SpaceTask
	err := s.getByID.GetContext(ctx, &task, id)
	if err != nil {
		return task, errors.Wrap(err, "failed to get space task")
	}
	return task, nil
}

func (s *spaceTaskDB) ListByBotSpaceID(ctx context.Context, botSpaceID string, status *string) ([]types.SpaceTask, error) {
	tasks := make([]types.SpaceTask, 0)
	var err error
	if status != nil {
		err = s.listByStatus.SelectContext(ctx, &tasks, botSpaceID, *status)
	} else {
		err = s.listByBotSpaceID.SelectContext(ctx, &tasks, botSpaceID)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to list space tasks")
	}
	return tasks, nil
}

func (s *spaceTaskDB) GetActiveByBotID(ctx context.Context, botSpaceID string, botID string) (*types.SpaceTask, error) {
	var task types.SpaceTask
	err := s.getActiveByBotID.GetContext(ctx, &task, botSpaceID, botID)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get active task for bot")
	}
	return &task, nil
}

func (s *spaceTaskDB) Update(ctx context.Context, task types.SpaceTask) (types.SpaceTask, error) {
	var result types.SpaceTask
	err := s.update.GetContext(ctx, &result, task)
	if err != nil {
		return result, errors.Wrap(err, "failed to update space task")
	}
	return result, nil
}
