package routes

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numbergroup/claw-swarm/pkg/types"
	ngerrors "github.com/numbergroup/errors"
	"github.com/numbergroup/server"
)

func (rh *RouteHandler) CreateTask(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireManagerBot(c)
	if !ok {
		return
	}

	var req types.CreateSpaceTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	task := types.SpaceTask{
		ID:             uuid.New().String(),
		BotSpaceID:     botSpaceID,
		Name:           req.Name,
		Description:    req.Description,
		Status:         "available",
		CreatedByBotID: claims.BotID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// If a bot is specified, assign it immediately
	if req.BotID != nil {
		targetBotID := *req.BotID
		bot, err := rh.botDB.GetByID(c, targetBotID)
		if err != nil {
			if ngerrors.Cause(err) == sql.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "target bot not found"})
				return
			}
			rh.log.WithError(err).Error("failed to get bot")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
			return
		}
		if bot.BotSpaceID != botSpaceID {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bot does not belong to this space"})
			return
		}

		activeTask, err := rh.spaceTaskDB.GetActiveByBotID(c, botSpaceID, targetBotID)
		if err != nil {
			rh.log.WithError(err).Error("failed to check active task")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
			return
		}
		if activeTask != nil {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error":       "bot already has an active task",
				"currentTask": activeTask,
			})
			return
		}

		task.BotID = &targetBotID
		task.Status = "in_progress"

		rh.updateBotStatusForTask(c, botSpaceID, targetBotID, bot.Name, claims.BotID, "Working on "+task.Name)
	}

	result, err := rh.spaceTaskDB.Insert(c, task)
	if err != nil {
		rh.log.WithError(err).Error("failed to insert task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (rh *RouteHandler) ListTasks(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can list tasks"})
		return
	}

	var status *string
	if claims.IsManager {
		if s := c.Query("status"); s != "" {
			status = &s
		}
	} else {
		available := "available"
		status = &available
	}

	tasks, err := rh.spaceTaskDB.ListByBotSpaceID(c, botSpaceID, status)
	if err != nil {
		rh.log.WithError(err).Error("failed to list tasks")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (rh *RouteHandler) GetCurrentTask(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can get their current task"})
		return
	}

	task, err := rh.spaceTaskDB.GetActiveByBotID(c, botSpaceID, claims.BotID)
	if err != nil {
		rh.log.WithError(err).Error("failed to get current task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get current task"})
		return
	}
	if task == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "no active task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (rh *RouteHandler) AcceptTask(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can accept tasks"})
		return
	}

	taskID, err := server.GetUUIDParam(c, "taskId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid taskId"})
		return
	}

	// Check bot doesn't already have an active task
	activeTask, err := rh.spaceTaskDB.GetActiveByBotID(c, botSpaceID, claims.BotID)
	if err != nil {
		rh.log.WithError(err).Error("failed to check active task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to accept task"})
		return
	}
	if activeTask != nil {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"error":       "you already have an active task",
			"currentTask": activeTask,
		})
		return
	}

	task, err := rh.spaceTaskDB.GetByID(c, taskID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to accept task"})
		return
	}

	if task.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.Status != "available" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "task is not available"})
		return
	}

	botID := claims.BotID
	now := time.Now()
	task.Status = "in_progress"
	task.BotID = &botID
	task.UpdatedAt = now

	result, err := rh.spaceTaskDB.Update(c, task)
	if err != nil {
		rh.log.WithError(err).Error("failed to update task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to accept task"})
		return
	}

	bot, err := rh.botDB.GetByID(c, claims.BotID)
	if err == nil {
		rh.updateBotStatusForTask(c, botSpaceID, claims.BotID, bot.Name, claims.BotID, "Working on "+task.Name)
	}

	c.JSON(http.StatusOK, result)
}

func (rh *RouteHandler) CompleteTask(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can complete tasks"})
		return
	}

	taskID, err := server.GetUUIDParam(c, "taskId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid taskId"})
		return
	}

	task, err := rh.spaceTaskDB.GetByID(c, taskID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to complete task"})
		return
	}

	if task.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.Status != "in_progress" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "task is not in progress"})
		return
	}
	if task.BotID == nil || *task.BotID != claims.BotID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you are not assigned to this task"})
		return
	}

	now := time.Now()
	task.Status = "completed"
	task.CompletedAt = &now
	task.UpdatedAt = now

	result, err := rh.spaceTaskDB.Update(c, task)
	if err != nil {
		rh.log.WithError(err).Error("failed to update task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to complete task"})
		return
	}

	bot, err := rh.botDB.GetByID(c, claims.BotID)
	if err == nil {
		rh.updateBotStatusForTask(c, botSpaceID, claims.BotID, bot.Name, claims.BotID, "")
	}

	c.JSON(http.StatusOK, result)
}

func (rh *RouteHandler) BlockTask(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	if !claims.IsBot {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only bots can block tasks"})
		return
	}

	taskID, err := server.GetUUIDParam(c, "taskId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid taskId"})
		return
	}

	task, err := rh.spaceTaskDB.GetByID(c, taskID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to block task"})
		return
	}

	if task.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.Status != "in_progress" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "task is not in progress"})
		return
	}
	if task.BotID == nil || *task.BotID != claims.BotID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you are not assigned to this task"})
		return
	}

	now := time.Now()
	task.Status = "blocked"
	task.UpdatedAt = now

	result, err := rh.spaceTaskDB.Update(c, task)
	if err != nil {
		rh.log.WithError(err).Error("failed to update task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to block task"})
		return
	}

	bot, err := rh.botDB.GetByID(c, claims.BotID)
	if err == nil {
		rh.updateBotStatusForTask(c, botSpaceID, claims.BotID, bot.Name, claims.BotID, "")
	}

	c.JSON(http.StatusOK, result)
}

func (rh *RouteHandler) AssignTask(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireManagerBot(c)
	if !ok {
		return
	}

	taskID, err := server.GetUUIDParam(c, "taskId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid taskId"})
		return
	}

	var req types.AssignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bot, err := rh.botDB.GetByID(c, req.BotID)
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "target bot not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get bot")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to assign task"})
		return
	}
	if bot.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bot does not belong to this space"})
		return
	}

	activeTask, err := rh.spaceTaskDB.GetActiveByBotID(c, botSpaceID, req.BotID)
	if err != nil {
		rh.log.WithError(err).Error("failed to check active task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to assign task"})
		return
	}
	if activeTask != nil {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"error":       "bot already has an active task",
			"currentTask": activeTask,
		})
		return
	}

	task, err := rh.spaceTaskDB.GetByID(c, taskID.String())
	if err != nil {
		if ngerrors.Cause(err) == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		rh.log.WithError(err).Error("failed to get task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to assign task"})
		return
	}

	if task.BotSpaceID != botSpaceID {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.Status != "available" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "task is not available"})
		return
	}

	now := time.Now()
	task.Status = "in_progress"
	task.BotID = &req.BotID
	task.UpdatedAt = now

	result, err := rh.spaceTaskDB.Update(c, task)
	if err != nil {
		rh.log.WithError(err).Error("failed to update task")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to assign task"})
		return
	}

	rh.updateBotStatusForTask(c, botSpaceID, req.BotID, bot.Name, claims.BotID, "Working on "+task.Name)

	c.JSON(http.StatusOK, result)
}

func (rh *RouteHandler) updateBotStatusForTask(c *gin.Context, botSpaceID, botID, botName, updatedByBotID, status string) {
	now := time.Now()
	botStatus := types.BotStatus{
		ID:             uuid.New().String(),
		BotSpaceID:     botSpaceID,
		BotID:          botID,
		BotName:        botName,
		Status:         status,
		UpdatedByBotID: updatedByBotID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if _, err := rh.botStatusDB.Upsert(c, botStatus); err != nil {
		rh.log.WithError(err).Error("failed to update bot status for task")
	}
}
