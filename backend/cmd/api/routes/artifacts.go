package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numbergroup/claw-swarm/pkg/types"
	"github.com/numbergroup/server"
)

func (rh *RouteHandler) CreateArtifact(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireManagerBot(c)
	if !ok {
		return
	}

	var req types.CreateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	artifact := types.Artifact{
		ID:             uuid.New().String(),
		BotSpaceID:     botSpaceID,
		Name:           req.Name,
		Description:    req.Description,
		Data:           req.Data,
		CreatedByBotID: claims.BotID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	result, err := rh.artifactDB.Insert(c, artifact)
	if err != nil {
		rh.log.WithError(err).Error("failed to insert artifact")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create artifact"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (rh *RouteHandler) ListArtifacts(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	limit, err := server.GetIntQuery(c, "limit", rh.conf.MaxMessagesPerPage, rh.conf.MaxMessagesPerPage)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var before *string
	if b := c.Query("before"); b != "" {
		before = &b
	}

	artifacts, err := rh.artifactDB.ListByBotSpaceID(c, botSpaceID, limit+1, before)
	if err != nil {
		rh.log.WithError(err).Error("failed to list artifacts")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list artifacts"})
		return
	}

	hasMore := len(artifacts) > limit
	if hasMore {
		artifacts = artifacts[:limit]
	}

	c.JSON(http.StatusOK, types.ArtifactListResponse{
		Artifacts: artifacts,
		Count:     len(artifacts),
		HasMore:   hasMore,
	})
}

func (rh *RouteHandler) DeleteArtifact(c *gin.Context) {
	_, _, ok := rh.requireManagerBot(c)
	if !ok {
		return
	}

	artifactID, err := server.GetUUIDParam(c, "artifactId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid artifactId"})
		return
	}

	if err := rh.artifactDB.Delete(c, artifactID.String()); err != nil {
		rh.log.WithError(err).Error("failed to delete artifact")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to delete artifact"})
		return
	}

	c.Status(http.StatusNoContent)
}
