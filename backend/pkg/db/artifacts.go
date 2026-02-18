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

type artifactDB struct {
	db               *sqlx.DB
	log              logrus.Ext1FieldLogger
	conf             *config.Config
	insert           *sqlx.NamedStmt
	listRecent       *sqlx.Stmt
	listBeforeCursor *sqlx.Stmt
	getCreatedAt     *sqlx.Stmt
	deleteStmt       *sqlx.Stmt
}

func NewArtifactDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (ArtifactDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.Artifact]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.Artifact]()

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO artifacts (%s) VALUES (:%s) RETURNING %s`,
		colStr, strings.Join(rawCols, ", :"), colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	listRecent, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM artifacts WHERE bot_space_id = $1 ORDER BY created_at DESC LIMIT $2`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listRecent statement")
	}

	listBeforeCursor, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM artifacts WHERE bot_space_id = $1 AND created_at < $2 ORDER BY created_at DESC LIMIT $3`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare listBeforeCursor statement")
	}

	getCreatedAt, err := sdb.PreparexContext(ctx,
		`SELECT created_at FROM artifacts WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getCreatedAt statement")
	}

	deleteStmt, err := sdb.PreparexContext(ctx, `DELETE FROM artifacts WHERE id = $1`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare delete statement")
	}

	return &artifactDB{
		db:               sdb,
		log:              conf.GetLogger(),
		conf:             conf,
		insert:           insert,
		listRecent:       listRecent,
		listBeforeCursor: listBeforeCursor,
		getCreatedAt:     getCreatedAt,
		deleteStmt:       deleteStmt,
	}, nil
}

func (a *artifactDB) Insert(ctx context.Context, artifact types.Artifact) (types.Artifact, error) {
	var result types.Artifact
	err := a.insert.GetContext(ctx, &result, artifact)
	if err != nil {
		return result, errors.Wrap(err, "failed to insert artifact")
	}
	return result, nil
}

func (a *artifactDB) ListByBotSpaceID(ctx context.Context, botSpaceID string, limit int, before *string) ([]types.Artifact, error) {
	artifacts := make([]types.Artifact, 0)

	if before == nil {
		err := a.listRecent.SelectContext(ctx, &artifacts, botSpaceID, limit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list recent artifacts")
		}
		return artifacts, nil
	}

	var cursorTime any
	err := a.getCreatedAt.GetContext(ctx, &cursorTime, *before)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cursor artifact created_at")
	}

	err = a.listBeforeCursor.SelectContext(ctx, &artifacts, botSpaceID, cursorTime, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list artifacts before cursor")
	}
	return artifacts, nil
}

func (a *artifactDB) Delete(ctx context.Context, id string) error {
	_, err := a.deleteStmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete artifact")
	}
	return nil
}
