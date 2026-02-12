package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/numbergroup/claw-swarm/cmd/api/routes"
	"github.com/numbergroup/claw-swarm/pkg/config"
	"github.com/numbergroup/claw-swarm/pkg/db"
	"github.com/numbergroup/claw-swarm/pkg/ws"
	"github.com/numbergroup/server"
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

	userDB, err := db.NewUserDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create user db")
	}

	botSpaceDB, err := db.NewBotSpaceDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create bot space db")
	}

	spaceMemberDB, err := db.NewSpaceMemberDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create space member db")
	}

	botDB, err := db.NewBotDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create bot db")
	}

	messageDB, err := db.NewMessageDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create message db")
	}

	botStatusDB, err := db.NewBotStatusDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create bot status db")
	}

	summaryDB, err := db.NewSummaryDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create summary db")
	}

	inviteCodeDB, err := db.NewInviteCodeDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create invite code db")
	}

	botSkillDB, err := db.NewBotSkillDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create bot skill db")
	}

	spaceTaskDB, err := db.NewSpaceTaskDB(ctx, conf, sdb)
	if err != nil {
		log.WithError(err).Fatal("failed to create space task db")
	}

	hub := ws.NewHub(log)

	rh := routes.NewRouteHandler(
		conf,
		userDB,
		botSpaceDB,
		spaceMemberDB,
		botDB,
		messageDB,
		botStatusDB,
		summaryDB,
		inviteCodeDB,
		botSkillDB,
		spaceTaskDB,
		hub,
	)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(server.CORSAllowAll)

	rh.ApplyRoutes(router)

	if err := server.ListenWithGracefulShutdown(ctx, log, router, conf.ServerConfig); err != nil {
		log.WithError(err).Fatal("server error")
	}
}
