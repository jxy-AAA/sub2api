package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type traceExportService interface {
	Export(ctx context.Context, now time.Time) (*service.TaodingTraceExportBundle, error)
}

type traceExportRootAdminReader interface {
	GetFirstAdmin(ctx context.Context) (*service.User, error)
}

type TraceExportHandler struct {
	traceService traceExportService
	userService  traceExportRootAdminReader
}

func NewTraceExportHandler(traceService *service.ModelInteractionTraceService, userService *service.UserService) *TraceExportHandler {
	return &TraceExportHandler{
		traceService: traceService,
		userService:  userService,
	}
}

func (h *TraceExportHandler) Export(c *gin.Context) {
	if !h.requireRootAdminJWT(c) {
		return
	}
	if h == nil || h.traceService == nil {
		response.InternalError(c, "trace export service is not configured")
		return
	}

	now := time.Now().UTC()
	bundle, err := h.traceService.Export(c.Request.Context(), now)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	body, err := json.Marshal(bundle.Records)
	if err != nil {
		response.InternalError(c, "failed to encode trace export")
		return
	}

	filename := fmt.Sprintf("sub2api-taoding-data-%s.json", now.Format("20060102150405"))
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Taoding-Trace-Export-Type", bundle.Type)
	c.Header("X-Taoding-Trace-Export-Version", fmt.Sprintf("%d", bundle.Version))
	c.Header("X-Taoding-Trace-Export-Count", fmt.Sprintf("%d", len(bundle.Records)))
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}

func (h *TraceExportHandler) requireRootAdminJWT(c *gin.Context) bool {
	if c.GetString("auth_method") != "jwt" {
		response.Forbidden(c, "root admin JWT required")
		return false
	}
	subject, ok := requireHumanAdminSubject(c, "root admin JWT required")
	if !ok {
		return false
	}
	if h == nil || h.userService == nil {
		response.InternalError(c, "root admin resolver is not configured")
		return false
	}
	firstAdmin, err := h.userService.GetFirstAdmin(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	if firstAdmin == nil || firstAdmin.ID != subject.UserID {
		response.Forbidden(c, "root admin access required")
		return false
	}
	return true
}
