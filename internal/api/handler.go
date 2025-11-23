package api

import (
	"io"
	"log/slog"
	"net/http"

	"aka-server/internal/aka"
	"aka-server/internal/config"
	"aka-server/internal/db"
	"aka-server/internal/model"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repo *db.Repository
	Cfg  *config.Config
}

func NewHandler(repo *db.Repository, cfg *config.Config) *Handler {
	return &Handler{Repo: repo, Cfg: cfg}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// Auth Vector Endpoint
	auth := v1.Group("/auth")
	auth.Use(IPAllowlist(h.Cfg.AuthAPIAllowedIPs))
	auth.POST("/:imsi", h.GenerateAuthVector)

	// Subscriber Management Endpoints
	subs := v1.Group("/subscribers")
	subs.Use(IPAllowlist(h.Cfg.DBAPIAllowedIPs))
	subs.POST("", h.CreateSubscriber)
	subs.GET("/:imsi", h.GetSubscriber)
	subs.PUT("/:imsi", h.UpdateSubscriber)
	subs.DELETE("/:imsi", h.DeleteSubscriber)
}

// Middleware for IP Allowlist
func IPAllowlist(allowedIPs []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(allowedIPs) == 0 {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		allowed := false
		for _, ip := range allowedIPs {
			if ip == clientIP {
				allowed = true
				break
			}
		}

		if !allowed {
			slog.Warn("Access denied", "ip", clientIP, "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.Next()
	}
}

type AuthRequest struct {
	Rand string `json:"rand"`
	Auts string `json:"auts"`
}

func (h *Handler) GenerateAuthVector(c *gin.Context) {
	imsi := c.Param("imsi")
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		// It's okay if body is empty, but if it's invalid JSON, error out.
		// Actually ShouldBindJSON returns error on empty body sometimes depending on content type.
		// We'll assume empty body is fine.
	}

	sub, err := h.Repo.GetSubscriber(c.Request.Context(), imsi)
	if err != nil {
		slog.Error("Database error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if sub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}

	var vec *aka.AuthVector
	var newSQN string

	if req.Rand != "" && req.Auts != "" {
		// Resync
		slog.Info("Processing Resync", "imsi", imsi)
		vec, newSQN, err = aka.Resync(sub, req.Rand, req.Auts)
	} else {
		// Normal Auth
		slog.Info("Processing Normal Auth", "imsi", imsi)
		vec, newSQN, err = aka.GenerateVector(sub)
	}

	if err != nil {
		slog.Error("AKA generation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update SQN in DB
	if err := h.Repo.UpdateSQN(c.Request.Context(), imsi, newSQN); err != nil {
		slog.Error("Failed to update SQN", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update SQN"})
		return
	}

	c.JSON(http.StatusOK, vec)
}

func (h *Handler) CreateSubscriber(c *gin.Context) {
	var sub model.Subscriber
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Repo.CreateSubscriber(c.Request.Context(), &sub); err != nil {
		slog.Error("Failed to create subscriber", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscriber"})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *Handler) GetSubscriber(c *gin.Context) {
	imsi := c.Param("imsi")
	sub, err := h.Repo.GetSubscriber(c.Request.Context(), imsi)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if sub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) UpdateSubscriber(c *gin.Context) {
	imsi := c.Param("imsi")
	var sub model.Subscriber
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub.IMSI = imsi // Ensure IMSI matches URL

	if err := h.Repo.UpdateSubscriber(c.Request.Context(), &sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscriber"})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteSubscriber(c *gin.Context) {
	imsi := c.Param("imsi")
	if err := h.Repo.DeleteSubscriber(c.Request.Context(), imsi); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subscriber"})
		return
	}
	c.Status(http.StatusNoContent)
}
