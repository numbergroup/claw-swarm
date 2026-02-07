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

type userDB struct {
	db             *sqlx.DB
	log            logrus.Ext1FieldLogger
	conf           *config.Config
	getByID        *sqlx.Stmt
	getByEmail     *sqlx.Stmt
	insert         *sqlx.NamedStmt
	updatePassword *sqlx.Stmt
}

func NewUserDB(ctx context.Context, conf *config.Config, sdb *sqlx.DB) (UserDB, error) {
	cols := psql.GetSQLColumnsQuoted[types.User]()
	colStr := strings.Join(cols, ", ")
	rawCols := psql.GetSQLColumns[types.User]()

	getByID, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM users WHERE id = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByID statement")
	}

	getByEmail, err := sdb.PreparexContext(ctx, fmt.Sprintf(
		`SELECT %s FROM users WHERE email = $1`, colStr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare getByEmail statement")
	}

	insert, err := sdb.PrepareNamedContext(ctx, fmt.Sprintf(
		`INSERT INTO users (%s) VALUES (:%s) RETURNING id`,
		colStr, strings.Join(rawCols, ", :")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare insert statement")
	}

	updatePassword, err := sdb.PreparexContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare updatePassword statement")
	}

	return &userDB{
		db:             sdb,
		log:            conf.GetLogger(),
		conf:           conf,
		getByID:        getByID,
		getByEmail:     getByEmail,
		insert:         insert,
		updatePassword: updatePassword,
	}, nil
}

func (u *userDB) GetByID(ctx context.Context, id string) (types.User, error) {
	var user types.User
	err := u.getByID.GetContext(ctx, &user, id)
	if err != nil {
		return user, errors.Wrap(err, "failed to get user by id")
	}
	return user, nil
}

func (u *userDB) GetByEmail(ctx context.Context, email string) (types.User, error) {
	var user types.User
	err := u.getByEmail.GetContext(ctx, &user, email)
	if err != nil {
		return user, errors.Wrap(err, "failed to get user by email")
	}
	return user, nil
}

func (u *userDB) Insert(ctx context.Context, user types.User) (string, error) {
	var id string
	err := u.insert.GetContext(ctx, &id, user)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert user")
	}
	return id, nil
}

func (u *userDB) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	_, err := u.updatePassword.ExecContext(ctx, passwordHash, id)
	if err != nil {
		return errors.Wrap(err, "failed to update password")
	}
	return nil
}
