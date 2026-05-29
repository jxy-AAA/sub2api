package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type traceCaptureAdminService interface {
	GetByID(ctx context.Context, id int64) (*service.ModelTraceCapture, error)
	List(ctx context.Context, filter service.ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*service.ModelTraceCapture, *pagination.PaginationResult, error)
	DeleteByID(ctx context.Context, id int64) (bool, error)
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
}

type traceRuleAdminService interface {
	Create(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error)
	Update(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error)
	GetByID(ctx context.Context, id int64) (*service.ModelTraceCaptureRule, error)
	List(ctx context.Context) ([]*service.ModelTraceCaptureRule, error)
	DeleteByID(ctx context.Context, id int64) (bool, error)
}

type traceExportTaskAdminService interface {
	ListTasks(ctx context.Context, params pagination.PaginationParams) ([]service.TraceExportTask, *pagination.PaginationResult, error)
	CreateTask(ctx context.Context, filters service.TraceExportTaskFilters, includeRaw bool, targetRecords int64, requestedBy int64) (*service.TraceExportTask, error)
	GetTask(ctx context.Context, id int64) (*service.TraceExportTask, error)
	CancelTask(ctx context.Context, id int64, canceledBy int64) error
	OpenDownload(ctx context.Context, id int64) (*service.TraceExportTaskDownload, error)
}

type TraceHandler struct {
	traceService      traceCaptureAdminService
	ruleService       traceRuleAdminService
	exportTaskService traceExportTaskAdminService
	rootAdminLookup   firstAdminLookup
}

type traceBatchDeleteRequest struct {
	IDs []int64 `json:"ids"`
}

type traceRuleCreateRequest struct {
	Name          string     `json:"name"`
	Enabled       *bool      `json:"enabled"`
	Priority      int        `json:"priority"`
	ModelPatterns []string   `json:"model_patterns"`
	UserIDs       []int64    `json:"user_ids"`
	APIKeyIDs     []int64    `json:"api_key_ids"`
	Keywords      []string   `json:"keywords"`
	MinTokens     *int64     `json:"min_tokens"`
	MaxTokens     *int64     `json:"max_tokens"`
	SamplingRatio float64    `json:"sampling_ratio"`
	ActiveFrom    *time.Time `json:"active_from"`
	ActiveTo      *time.Time `json:"active_to"`
}

type traceRuleUpdateRequest struct {
	Name          *string    `json:"name"`
	Enabled       *bool      `json:"enabled"`
	Priority      *int       `json:"priority"`
	ModelPatterns []string   `json:"model_patterns"`
	UserIDs       []int64    `json:"user_ids"`
	APIKeyIDs     []int64    `json:"api_key_ids"`
	Keywords      []string   `json:"keywords"`
	MinTokens     *int64     `json:"min_tokens"`
	MaxTokens     *int64     `json:"max_tokens"`
	SamplingRatio *float64   `json:"sampling_ratio"`
	ActiveFrom    *time.Time `json:"active_from"`
	ActiveTo      *time.Time `json:"active_to"`
}

type traceFilterRequest struct {
	Model           string `json:"model"`
	UserID          *int64 `json:"user_id"`
	APIKeyID        *int64 `json:"api_key_id"`
	CaptureRuleID   *int64 `json:"capture_rule_id"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	Timezone        string `json:"timezone"`
	Keyword         string `json:"keyword"`
	MinInputTokens  *int64 `json:"min_input_tokens"`
	MaxInputTokens  *int64 `json:"max_input_tokens"`
	MinOutputTokens *int64 `json:"min_output_tokens"`
	MaxOutputTokens *int64 `json:"max_output_tokens"`
	MinTotalTokens  *int64 `json:"min_total_tokens"`
	MaxTotalTokens  *int64 `json:"max_total_tokens"`
}

type traceExportTaskCreateRequest struct {
	Filters       traceFilterRequest `json:"filters"`
	IncludeRaw    bool               `json:"include_raw"`
	TargetRecords *int64             `json:"target_records"`
}

func NewTraceHandler(
	traceService *service.ModelTraceCaptureService,
	ruleService *service.ModelTraceCaptureRuleService,
	exportTaskService *service.TraceExportTaskService,
	userService *service.UserService,
) *TraceHandler {
	return &TraceHandler{
		traceService:      traceService,
		ruleService:       ruleService,
		exportTaskService: exportTaskService,
		rootAdminLookup:   userService,
	}
}

func (h *TraceHandler) List(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace access"); !ok {
		return
	}
	if h == nil || h.traceService == nil {
		response.InternalError(c, "trace capture service is not configured")
		return
	}

	page, pageSize := response.ParsePagination(c)
	filters, err := traceFiltersFromQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	items, result, err := h.traceService.List(c.Request.Context(), filters.ToModelTraceCaptureListFilter(), pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	total := int64(len(items))
	if result != nil {
		total = result.Total
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *TraceHandler) GetByID(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace access"); !ok {
		return
	}
	if h == nil || h.traceService == nil {
		response.InternalError(c, "trace capture service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	item, err := h.traceService.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "Trace not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *TraceHandler) Delete(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace deletion"); !ok {
		return
	}
	if h == nil || h.traceService == nil {
		response.InternalError(c, "trace capture service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	deleted, err := h.traceService.DeleteByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !deleted {
		response.NotFound(c, "Trace not found")
		return
	}

	response.Success(c, gin.H{"id": id, "deleted": true})
}

func (h *TraceHandler) BatchDelete(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace deletion"); !ok {
		return
	}
	if h == nil || h.traceService == nil {
		response.InternalError(c, "trace capture service is not configured")
		return
	}

	var req traceBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	ids := normalizeTraceBatchDeleteIDs(req.IDs)
	if len(ids) == 0 {
		response.BadRequest(c, "ids must contain at least one positive id")
		return
	}

	deletedCount, err := h.traceService.DeleteByIDs(c.Request.Context(), ids)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted_count": deletedCount})
}

func (h *TraceHandler) ListRules(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace rule access"); !ok {
		return
	}
	if h == nil || h.ruleService == nil {
		response.InternalError(c, "trace rule service is not configured")
		return
	}

	items, err := h.ruleService.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *TraceHandler) GetRuleByID(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace rule access"); !ok {
		return
	}
	if h == nil || h.ruleService == nil {
		response.InternalError(c, "trace rule service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "Trace rule not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rule)
}

func (h *TraceHandler) CreateRule(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace rule updates"); !ok {
		return
	}
	if h == nil || h.ruleService == nil {
		response.InternalError(c, "trace rule service is not configured")
		return
	}

	var req traceRuleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.ModelTraceCaptureRule{
		Name:          req.Name,
		Enabled:       true,
		Priority:      req.Priority,
		ModelPatterns: req.ModelPatterns,
		UserIDs:       req.UserIDs,
		APIKeyIDs:     req.APIKeyIDs,
		Keywords:      req.Keywords,
		MinTokens:     req.MinTokens,
		MaxTokens:     req.MaxTokens,
		SamplingRatio: req.SamplingRatio,
		ActiveFrom:    req.ActiveFrom,
		ActiveTo:      req.ActiveTo,
	}
	if req.Enabled != nil {
		input.Enabled = *req.Enabled
	}
	if err := input.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	rule, err := h.ruleService.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rule)
}

func (h *TraceHandler) UpdateRule(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace rule updates"); !ok {
		return
	}
	if h == nil || h.ruleService == nil {
		response.InternalError(c, "trace rule service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req traceRuleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	existing, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "Trace rule not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}

	rule := &service.ModelTraceCaptureRule{
		ID:            id,
		Name:          existing.Name,
		Enabled:       existing.Enabled,
		Priority:      existing.Priority,
		ModelPatterns: append([]string(nil), existing.ModelPatterns...),
		UserIDs:       append([]int64(nil), existing.UserIDs...),
		APIKeyIDs:     append([]int64(nil), existing.APIKeyIDs...),
		Keywords:      append([]string(nil), existing.Keywords...),
		MinTokens:     cloneTraceRuleInt64Ptr(existing.MinTokens),
		MaxTokens:     cloneTraceRuleInt64Ptr(existing.MaxTokens),
		SamplingRatio: existing.SamplingRatio,
		ActiveFrom:    cloneTraceRuleTimePtr(existing.ActiveFrom),
		ActiveTo:      cloneTraceRuleTimePtr(existing.ActiveTo),
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.ModelPatterns != nil {
		rule.ModelPatterns = req.ModelPatterns
	}
	if req.UserIDs != nil {
		rule.UserIDs = req.UserIDs
	}
	if req.APIKeyIDs != nil {
		rule.APIKeyIDs = req.APIKeyIDs
	}
	if req.Keywords != nil {
		rule.Keywords = req.Keywords
	}
	if req.MinTokens != nil {
		rule.MinTokens = req.MinTokens
	}
	if req.MaxTokens != nil {
		rule.MaxTokens = req.MaxTokens
	}
	if req.SamplingRatio != nil {
		rule.SamplingRatio = *req.SamplingRatio
	}
	if req.ActiveFrom != nil {
		rule.ActiveFrom = req.ActiveFrom
	}
	if req.ActiveTo != nil {
		rule.ActiveTo = req.ActiveTo
	}
	if err := rule.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.ruleService.Update(c.Request.Context(), rule)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *TraceHandler) DeleteRule(c *gin.Context) {
	if _, ok := requireHumanAdminSubject(c, "admin JWT required for trace rule updates"); !ok {
		return
	}
	if h == nil || h.ruleService == nil {
		response.InternalError(c, "trace rule service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	deleted, err := h.ruleService.DeleteByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !deleted {
		response.NotFound(c, "Trace rule not found")
		return
	}

	response.Success(c, gin.H{"message": "Trace rule deleted successfully"})
}

func (h *TraceHandler) ListExportTasks(c *gin.Context) {
	if _, ok := h.requireRootAdmin(c); !ok {
		return
	}
	if h == nil || h.exportTaskService == nil {
		response.InternalError(c, "trace export task service is not configured")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, result, err := h.exportTaskService.ListTasks(c.Request.Context(), pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	total := int64(len(items))
	if result != nil {
		total = result.Total
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *TraceHandler) CreateExportTask(c *gin.Context) {
	requestedBy, ok := h.requireRootAdmin(c)
	if !ok {
		return
	}
	if h == nil || h.exportTaskService == nil {
		response.InternalError(c, "trace export task service is not configured")
		return
	}

	var req traceExportTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	filters, err := req.Filters.toTaskFilters()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	targetRecords := service.TraceExportTaskDefaultTargetRecords
	if req.TargetRecords != nil {
		if *req.TargetRecords <= 0 {
			response.BadRequest(c, "target_records must be > 0")
			return
		}
		targetRecords = *req.TargetRecords
	}

	task, err := h.exportTaskService.CreateTask(c.Request.Context(), filters, req.IncludeRaw, targetRecords, requestedBy)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Accepted(c, task)
}

func (h *TraceHandler) GetExportTask(c *gin.Context) {
	if _, ok := h.requireRootAdmin(c); !ok {
		return
	}
	if h == nil || h.exportTaskService == nil {
		response.InternalError(c, "trace export task service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	task, err := h.exportTaskService.GetTask(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, task)
}

func (h *TraceHandler) CancelExportTask(c *gin.Context) {
	canceledBy, ok := h.requireRootAdmin(c)
	if !ok {
		return
	}
	if h == nil || h.exportTaskService == nil {
		response.InternalError(c, "trace export task service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if err := h.exportTaskService.CancelTask(c.Request.Context(), id, canceledBy); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "status": service.TraceExportTaskStatusCanceled})
}

func (h *TraceHandler) DownloadExportTask(c *gin.Context) {
	if _, ok := h.requireRootAdmin(c); !ok {
		return
	}
	if h == nil || h.exportTaskService == nil {
		response.InternalError(c, "trace export task service is not configured")
		return
	}

	id, err := parseAdminPositiveInt64(c.Param("id"), "id")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	download, err := h.exportTaskService.OpenDownload(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = download.Reader.Close() }()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, download.Filename))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Trace-Export-Task-ID", strconv.FormatInt(download.Task.ID, 10))
	c.DataFromReader(http.StatusOK, download.Size, download.ContentType, download.Reader, nil)
}

func (h *TraceHandler) requireRootAdmin(c *gin.Context) (int64, bool) {
	subject, ok := requireHumanAdminSubject(c, "root admin JWT required")
	if !ok {
		return 0, false
	}
	if err := rejectAdminAPIKey(c, "ROOT_ADMIN_JWT_REQUIRED", "root admin JWT required"); err != nil {
		response.ErrorFrom(c, err)
		return 0, false
	}
	if h == nil || h.rootAdminLookup == nil {
		response.InternalError(c, "root admin resolver is not configured")
		return 0, false
	}

	firstAdmin, err := h.rootAdminLookup.GetFirstAdmin(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return 0, false
	}
	if firstAdmin == nil || firstAdmin.ID != subject.UserID {
		response.Forbidden(c, "root admin access required")
		return 0, false
	}
	return subject.UserID, true
}

func traceFiltersFromQuery(c *gin.Context) (service.TraceExportTaskFilters, error) {
	req := traceFilterRequest{
		Model:     c.Query("model"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Timezone:  c.Query("timezone"),
		Keyword:   c.Query("keyword"),
	}

	var err error
	if req.UserID, err = parseTraceOptionalPositiveInt64(c.Query("user_id"), "user_id"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.APIKeyID, err = parseTraceOptionalPositiveInt64(c.Query("api_key_id"), "api_key_id"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.CaptureRuleID, err = parseTraceOptionalPositiveInt64(c.Query("capture_rule_id"), "capture_rule_id"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.MinInputTokens, err = parseTraceOptionalNonNegativeInt64(c.Query("min_input_tokens"), "min_input_tokens"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.MaxInputTokens, err = parseTraceOptionalNonNegativeInt64(c.Query("max_input_tokens"), "max_input_tokens"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.MinOutputTokens, err = parseTraceOptionalNonNegativeInt64(c.Query("min_output_tokens"), "min_output_tokens"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.MaxOutputTokens, err = parseTraceOptionalNonNegativeInt64(c.Query("max_output_tokens"), "max_output_tokens"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.MinTotalTokens, err = parseTraceOptionalNonNegativeInt64(c.Query("min_total_tokens"), "min_total_tokens"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	if req.MaxTotalTokens, err = parseTraceOptionalNonNegativeInt64(c.Query("max_total_tokens"), "max_total_tokens"); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	return req.toTaskFilters()
}

func (r traceFilterRequest) toTaskFilters() (service.TraceExportTaskFilters, error) {
	startTime, err := parseTraceStartBoundary(r.StartTime, r.StartDate, r.Timezone)
	if err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	endTime, err := parseTraceEndBoundary(r.EndTime, r.EndDate, r.Timezone)
	if err != nil {
		return service.TraceExportTaskFilters{}, err
	}

	filters := service.TraceExportTaskFilters{
		Model:           strings.TrimSpace(r.Model),
		UserID:          cloneTraceRuleInt64Ptr(r.UserID),
		APIKeyID:        cloneTraceRuleInt64Ptr(r.APIKeyID),
		CaptureRuleID:   cloneTraceRuleInt64Ptr(r.CaptureRuleID),
		StartTime:       startTime,
		EndTime:         endTime,
		Keyword:         strings.TrimSpace(r.Keyword),
		MinInputTokens:  cloneTraceRuleInt64Ptr(r.MinInputTokens),
		MaxInputTokens:  cloneTraceRuleInt64Ptr(r.MaxInputTokens),
		MinOutputTokens: cloneTraceRuleInt64Ptr(r.MinOutputTokens),
		MaxOutputTokens: cloneTraceRuleInt64Ptr(r.MaxOutputTokens),
		MinTotalTokens:  cloneTraceRuleInt64Ptr(r.MinTotalTokens),
		MaxTotalTokens:  cloneTraceRuleInt64Ptr(r.MaxTotalTokens),
	}
	filters.Normalize()
	if err := filters.Validate(); err != nil {
		return service.TraceExportTaskFilters{}, err
	}
	return filters, nil
}

func parseTraceOptionalPositiveInt64(raw string, field string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, fmt.Errorf("invalid %s", field)
	}
	return &value, nil
}

func parseTraceOptionalNonNegativeInt64(raw string, field string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return nil, fmt.Errorf("invalid %s", field)
	}
	return &value, nil
}

func parseTraceStartBoundary(startTimeRaw, startDateRaw, userTZ string) (*time.Time, error) {
	startTimeRaw = strings.TrimSpace(startTimeRaw)
	if startTimeRaw != "" {
		parsed, err := time.Parse(time.RFC3339, startTimeRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time, use RFC3339")
		}
		return &parsed, nil
	}

	startDateRaw = strings.TrimSpace(startDateRaw)
	if startDateRaw == "" {
		return nil, nil
	}

	parsed, err := timezone.ParseInUserLocation("2006-01-02", startDateRaw, userTZ)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date, use YYYY-MM-DD")
	}
	return &parsed, nil
}

func parseTraceEndBoundary(endTimeRaw, endDateRaw, userTZ string) (*time.Time, error) {
	endTimeRaw = strings.TrimSpace(endTimeRaw)
	if endTimeRaw != "" {
		parsed, err := time.Parse(time.RFC3339, endTimeRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time, use RFC3339")
		}
		return &parsed, nil
	}

	endDateRaw = strings.TrimSpace(endDateRaw)
	if endDateRaw == "" {
		return nil, nil
	}

	parsed, err := timezone.ParseInUserLocation("2006-01-02", endDateRaw, userTZ)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date, use YYYY-MM-DD")
	}
	parsed = parsed.AddDate(0, 0, 1)
	return &parsed, nil
}

func cloneTraceRuleInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTraceRuleTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func normalizeTraceBatchDeleteIDs(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
