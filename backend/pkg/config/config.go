package config

import (
	"context"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/innodv/psql"
	"github.com/jmoiron/sqlx"
	"github.com/numbergroup/cleanenv"
	"github.com/numbergroup/config"
	"github.com/numbergroup/config/gcp"
	"github.com/numbergroup/server"
)

type Config struct {
	config.BaseConfig
	ServerConfig       server.Config
	PSQL               psql.Config
	Secrets            []string      `env:"SECRETS" env-default:""`
	JWTSecret          string        `env:"JWT_SECRET"`
	JWTExpiration      time.Duration `env:"JWT_EXPIRATION" env-default:"24h"`
	MaxBodySize        int64         `env:"MAX_BODY_SIZE" env-default:"1048576"`
	MaxMessagesPerPage int           `env:"MAX_MESSAGES_PER_PAGE" env-default:"30"`
	MaxMessageLength   int           `env:"MAX_MESSAGE_LENGTH" env-default:"10000"`
}

func (c Config) ConnectPSQL(ctx context.Context) (*sqlx.DB, error) {
	dbConn, err := psql.OpenConnectionPool(c.PSQL, c.GetLogger())
	if err != nil {
		return nil, err
	}
	err = dbConn.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return dbConn, nil
}

func NewConfig(ctx context.Context) (*Config, error) {
	conf := &Config{}
	err := cleanenv.ReadEnv(conf)
	if err != nil {
		return nil, err
	}

	conf.ServerConfig, err = server.LoadServerConfigFromEnv()
	if err != nil {
		return nil, err
	}

	if len(conf.Secrets) != 0 {
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			return nil, err
		}
		defer client.Close()

		err = gcp.LoadJSONSecretsIntoEnvThenUpdateConfig(ctx, client, conf.Secrets, conf)
		if err != nil {
			return nil, err
		}
	}

	conf.PSQL, err = psql.NewConfig()
	if err != nil {
		return nil, err
	}

	return conf, nil
}
