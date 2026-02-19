package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/numbergroup/claw-swarm/pkg/types"
	"github.com/numbergroup/server"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (rh *RouteHandler) PostMessage(c *gin.Context) {
	claims, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	var req types.PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Content) > rh.conf.MaxMessageLength {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "message too long"})
		return
	}

	var senderID, senderName, senderType string
	if claims.IsBot {
		senderID = claims.BotID
		senderType = "bot"
		bot, err := rh.botDB.GetByID(c, claims.BotID)
		if err != nil {
			rh.log.WithError(err).Error("failed to get bot for sender name")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to post message"})
			return
		}
		if bot.IsMuted {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bot is muted"})
			return
		}
		senderName = bot.Name
	} else {
		senderID = claims.UserID
		senderType = "user"
		user, err := rh.userDB.GetByID(c, claims.UserID)
		if err != nil {
			rh.log.WithError(err).Error("failed to get user for sender name")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to post message"})
			return
		}
		if user.DisplayName != nil {
			senderName = *user.DisplayName
		} else {
			senderName = user.Email
		}
	}

	msg := types.Message{
		ID:         uuid.New().String(),
		BotSpaceID: botSpaceID,
		SenderID:   senderID,
		SenderName: senderName,
		SenderType: senderType,
		Content:    req.Content,
		CreatedAt:  time.Now(),
	}

	_, err := rh.messageDB.Insert(c, msg)
	if err != nil {
		rh.log.WithError(err).Error("failed to insert message")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to post message"})
		return
	}

	data, err := json.Marshal(msg)
	if err == nil {
		rh.hub.Broadcast(botSpaceID, data)
	}

	c.JSON(http.StatusCreated, msg)
}

func (rh *RouteHandler) ListMessages(c *gin.Context) {
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

	messages, err := rh.messageDB.ListByBotSpaceID(c, botSpaceID, limit+1, before)
	if err != nil {
		rh.log.WithError(err).Error("failed to list messages")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list messages"})
		return
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	c.JSON(http.StatusOK, types.MessageListResponse{
		Messages: messages,
		Count:    len(messages),
		HasMore:  hasMore,
	})
}

func (rh *RouteHandler) GetMessagesSince(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	messageID, err := server.GetUUIDParam(c, "messageId")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid messageId"})
		return
	}

	limit, err := server.GetIntQuery(c, "limit", rh.conf.MaxMessagesPerPage, rh.conf.MaxMessagesPerPage)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messages, err := rh.messageDB.ListSince(c, botSpaceID, messageID.String(), limit+1)
	if err != nil {
		rh.log.WithError(err).Error("failed to list messages since")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	c.JSON(http.StatusOK, types.MessageListResponse{
		Messages: messages,
		Count:    len(messages),
		HasMore:  hasMore,
	})
}

func (rh *RouteHandler) SubscribeMessages(c *gin.Context) {
	_, botSpaceID, ok := rh.requireSpaceAccess(c)
	if !ok {
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		rh.log.WithError(err).Error("failed to upgrade websocket")
		return
	}

	client := rh.hub.NewClient(conn, botSpaceID)
	rh.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
