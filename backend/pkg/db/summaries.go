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

type summaryDB struct {
	db             *sqlx.DB
	log            logrus.Ext1FieldLogger
	conf           *config.Config
	getByBotSpaceID *sqlx.Stmt
	upsert          *sqlx.NamedStmt
}

func NewSummaryDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (SummaryDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.Summary]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.Summary]()

	getByBotSpaceID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM summaries WHERE bot_space_id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByBotSpaceID statement")
	}

	upsert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO summaries (%s) VALUES (:%s)
		ON CONFLICT (bot_space_id)
		DO UPDATE SET content = EXCLUDED.content, created_by_bot_id = EXCLUDED.created_by_bot_id,
		             updated_at = now()
		RETURNING %s`,
		colStr, strings.Join(rawCols, ", :"), colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare upsert statement")
	}

	return &summaryDB{
		db:              sdb,
		log:             conf.GetLogger(),
		conf:            conf,
		getByBotSpaceID: getByBotSpaceID,
		upsert:          upsert,
	}, nil
}

func (s *summaryDB) GetByBotSpaceID(ctx context.Context, botSpaceID string) (types.Summary, error) {
	var summary types.Summary
	err := s.getByBotSpaceID.GetContext(ctx, &summary, botSpaceID)
	if err != nil {
		return summary, errors.Wrap(err, "failed to get summary by bot space id")
	}
	return summary, nil
}

func (s *summaryDB) Upsert(ctx context.Context, summary types.Summary) (types.Summary, error) {
	var result types.Summary
	err := s.upsert.GetContext(ctx, &result, summary)
	if err != nil {
		return result, errors.Wrap(err, "failed to upsert summary")
	}
	return result, nil
}
