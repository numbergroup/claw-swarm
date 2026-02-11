package main

import (
	"context"

	"github.com/numbergroup/claw-swarm/pkg/config"
	"github.com/numbergroup/claw-swarm/pkg/db"
)

func main() {
	ctx := context.Background()

	conf, err := config.NewConfig(ctx)
	if err != nil {
		panic(err)
	}

	log := conf.GetLogger()

	sdb, err := conf.ConnectPSQL(ctx)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to database")
	}
	defer sdb.Close()

	messageDB, err := db.NewMessageDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create message db")
	}

	spaceIDs, err := messageDB.ListSpaceIDsExceedingCount(ctx, conf.MaxMessagesPerSpace)
	if err != nil {
		log.WithError(err).Fatal("failed to list spaces exceeding message limit")
	}

	if len(spaceIDs) == 0 {
		log.Info("no spaces exceed the message limit")
		return
	}

	log.WithField("count", len(spaceIDs)).Info("found spaces exceeding message limit")

	var totalDeleted int64
	for _, spaceID := range spaceIDs {
		deleted, err := messageDB.DeleteOlderThanNth(ctx, spaceID, conf.MaxMessagesPerSpace)
		if err != nil {
			log.WithError(err).WithField("spaceID", spaceID).Error("failed to delete old messages")
			continue
		}
		totalDeleted += deleted
		log.WithField("spaceID", spaceID).WithField("deleted", deleted).Info("cleaned up messages")
	}

	log.WithField("totalDeleted", totalDeleted).WithField("spacesProcessed", len(spaceIDs)).Info("cleanup complete")
}
