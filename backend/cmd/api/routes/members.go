package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/numbergroup/server"
)

func (rh *RouteHandler) ListMembers(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	members, err := rh.spaceMemberDB.ListByBotSpaceID(c, botSpaceID)
	if err != nil {
		rh.log.WithError(err).Error("failed to list members")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}

	c.JSON(http.StatusOK, members)
}

func (rh *RouteHandler) RemoveMember(c *gin.Context) {
	_, botSpaceID, ok := rh.requireOwner(c)
	if !ok {
		return
	}

	userID, err := server.GetUUIDParam(c, "userId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid userId"})
		return
	}

	if err := rh.spaceMemberDB.Delete(c, botSpaceID, userID.String()); err != nil {
		rh.log.WithError(err).Error("failed to remove member")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}

	c.Status(http.StatusNoContent)
}
