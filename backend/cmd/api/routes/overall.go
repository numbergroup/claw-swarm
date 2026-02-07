package routes

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/numbergroup/claw-swarm/pkg/types"
	ngerrors "github.com/numbergroup/errors"
	"github.com/numbergroup/server"
)

func (rh *RouteHandler) GetOverall(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	limit, err := server.GetIntQuery(c, "limit", rh.conf.MaxMessagesPerPage, rh.conf.MaxMessagesPerPage)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messages, err := rh.messageDB.ListByBotSpaceID(c, botSpaceID, limit+1, nil)
	if err != nil {
		rh.log.WithError(err).Error("failed to list messages")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get overall"})
		return
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	resp := types.OverallResponse{
		Messages: types.MessageListResponse{
			Messages: messages,
			Count:    len(messages),
			HasMore:  hasMore,
		},
	}

	summary, err := rh.summaryDB.GetByBotSpaceID(c, botSpaceID)
	if err != nil {
		if ngerrors.Cause(err) != sql.ErrNoRows {
			rh.log.WithError(err).Error("failed to get summary")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get overall"})
			return
		}
	} else {
		resp.Summary = &summary
	}

	c.JSON(http.StatusOK, resp)
}
