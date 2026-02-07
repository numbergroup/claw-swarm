package routes

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	ngerrors "github.com/numbergroup/errors"
	"github.com/numbergroup/server"
)

func (rh *RouteHandler) ListBots(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	bots, err := rh.botDB.ListByBotSpaceID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to list bots")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list bots"})
		return
	}

	c.JSON(http.StatusOK, bots)
}

func (rh *RouteHandler) GetBot(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	botID, err := server.GetUUIDParam(c, "botId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botId"})
		return
	}

	bot, err := rh.botDB.GetByID(c, botID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get bot"})
		return
	}

	if bot.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}

	c.JSON(http.StatusOK, bot)
}

func (rh *RouteHandler) RemoveBot(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	botID, err := server.GetUUIDParam(c, "botId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botId"})
		return
	}

	bot, err := rh.botDB.GetByID(c, botID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to remove bot"})
		return
	}

	if bot.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}

	if bot.IsManager {
		if err := rh.botSpaceDB.ClearManagerBotID(c, botSpaceID); err != nil {
			rh.log.WithError(err).Error("failed to clear manager bot id")
		}
	}

	if err := rh.botDB.Delete(c, botID.String()); err != nil {
		rh.log.WithError(err).Error("failed to delete bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to remove bot"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (rh *RouteHandler) AssignManager(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	botID, err := server.GetUUIDParam(c, "botId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botId"})
		return
	}

	bot, err := rh.botDB.GetByID(c, botID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to assign manager"})
		return
	}

	if bot.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}

	if err := rh.botDB.SetManager(c, botID.String(), true); err != nil {
		rh.log.WithError(err).Error("failed to set manager")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to assign manager"})
		return
	}

	if err := rh.botSpaceDB.SetManagerBotID(c, botSpaceID, botID.String()); err != nil {
		rh.log.WithError(err).Error("failed to set manager bot id on space")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to assign manager"})
		return
	}

	bot.IsManager = true
	c.JSON(http.StatusOK, bot)
}

func (rh *RouteHandler) RemoveManagerRole(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	botID, err := server.GetUUIDParam(c, "botId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid botId"})
		return
	}

	bot, err := rh.botDB.GetByID(c, botID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to remove manager"})
		return
	}

	if bot.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}

	if err := rh.botDB.SetManager(c, botID.String(), false); err != nil {
		rh.log.WithError(err).Error("failed to unset manager")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to remove manager"})
		return
	}

	if err := rh.botSpaceDB.ClearManagerBotID(c, botSpaceID); err != nil {
		rh.log.WithError(err).Error("failed to clear manager bot id")
	}

	c.Status(http.StatusNoContent)
}
