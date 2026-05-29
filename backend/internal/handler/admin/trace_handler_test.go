package admin

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type traceCaptureServiceStub struct {
	listItems  []*service.ModelTraceCapture
	listResult *pagination.PaginationResult
	listErr    error
	lastFilter service.ModelTraceCaptureListFilter
	getItem    *service.ModelTraceCapture
	getErr     error
	deleteOK   bool
	deleteErr  error
	deleteIDs  []int64
}

func (s *traceCaptureServiceStub) GetByID(ctx context.Context, id int64) (*service.ModelTraceCapture, error) {
	return s.getItem, s.getErr
}

func (s *traceCaptureServiceStub) List(ctx context.Context, filter service.ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*service.ModelTraceCapture, *pagination.PaginationResult, error) {
	s.lastFilter = filter
	return s.listItems, s.listResult, s.listErr
}

func (s *traceCaptureServiceStub) DeleteByID(ctx context.Context, id int64) (bool, error) {
	return s.deleteOK, s.deleteErr
}

func (s *traceCaptureServiceStub) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	s.deleteIDs = append([]int64(nil), ids...)
	return int64(len(ids)), s.deleteErr
}

type traceRuleServiceStub struct {
	createInput *service.ModelTraceCaptureRule
	createRule  *service.ModelTraceCaptureRule
	createErr   error
	getRule     *service.ModelTraceCaptureRule
	getErr      error
	listRules   []*service.ModelTraceCaptureRule
	listErr     error
	updateInput *service.ModelTraceCaptureRule
	updateRule  *service.ModelTraceCaptureRule
	updateErr   error
	deleteOK    bool
	deleteErr   error
}

func (s *traceRuleServiceStub) Create(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error) {
	s.createInput = rule
	return s.createRule, s.createErr
}

func (s *traceRuleServiceStub) Update(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error) {
	s.updateInput = rule
	return s.updateRule, s.updateErr
}

func (s *traceRuleServiceStub) GetByID(ctx context.Context, id int64) (*service.ModelTraceCaptureRule, error) {
	return s.getRule, s.getErr
}

func (s *traceRuleServiceStub) List(ctx context.Context) ([]*service.ModelTraceCaptureRule, error) {
	return s.listRules, s.listErr
}

func (s *traceRuleServiceStub) DeleteByID(ctx context.Context, id int64) (bool, error) {
	return s.deleteOK, s.deleteErr
}

type traceExportTaskServiceStub struct {
	createTask        *service.TraceExportTask
	createErr         error
	lastCreateFilters service.TraceExportTaskFilters
	lastIncludeRaw    bool
	lastTargetRecords int64
	lastRequestedBy   int64
	getTask           *service.TraceExportTask
	getErr            error
	cancelErr         error
	lastCanceledBy    int64
	listTasks         []service.TraceExportTask
	listResult        *pagination.PaginationResult
	listErr           error
	download          *service.TraceExportTaskDownload
	downloadErr       error
}

func (s *traceExportTaskServiceStub) ListTasks(ctx context.Context, params pagination.PaginationParams) ([]service.TraceExportTask, *pagination.PaginationResult, error) {
	return s.listTasks, s.listResult, s.listErr
}

func (s *traceExportTaskServiceStub) CreateTask(ctx context.Context, filters service.TraceExportTaskFilters, includeRaw bool, targetRecords int64, requestedBy int64) (*service.TraceExportTask, error) {
	s.lastCreateFilters = filters
	s.lastIncludeRaw = includeRaw
	s.lastTargetRecords = targetRecords
	s.lastRequestedBy = requestedBy
	return s.createTask, s.createErr
}

func (s *traceExportTaskServiceStub) GetTask(ctx context.Context, id int64) (*service.TraceExportTask, error) {
	return s.getTask, s.getErr
}

func (s *traceExportTaskServiceStub) CancelTask(ctx context.Context, id int64, canceledBy int64) error {
	s.lastCanceledBy = canceledBy
	return s.cancelErr
}

func (s *traceExportTaskServiceStub) OpenDownload(ctx context.Context, id int64) (*service.TraceExportTaskDownload, error) {
	return s.download, s.downloadErr
}

type rootAdminLookupStub struct {
	user *service.User
	err  error
}

func (s *rootAdminLookupStub) GetFirstAdmin(context.Context) (*service.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func setupTraceRouter(handler *TraceHandler, authMethod string, subject *middleware.AuthSubject) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if subject != nil {
		authSubject := *subject
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUser), authSubject)
			if authMethod != "" {
				c.Set("auth_method", authMethod)
			}
			c.Next()
		})
	}

	router.GET("/api/v1/admin/traces", handler.List)
	router.GET("/api/v1/admin/traces/:id", handler.GetByID)
	router.DELETE("/api/v1/admin/traces/:id", handler.Delete)
	router.POST("/api/v1/admin/traces/batch-delete", handler.BatchDelete)
	router.GET("/api/v1/admin/traces/rules", handler.ListRules)
	router.GET("/api/v1/admin/traces/rules/:id", handler.GetRuleByID)
	router.POST("/api/v1/admin/traces/rules", handler.CreateRule)
	router.PUT("/api/v1/admin/traces/rules/:id", handler.UpdateRule)
	router.DELETE("/api/v1/admin/traces/rules/:id", handler.DeleteRule)
	router.GET("/api/v1/admin/traces/export-tasks", handler.ListExportTasks)
	router.POST("/api/v1/admin/traces/export-tasks", handler.CreateExportTask)
	router.GET("/api/v1/admin/traces/export-tasks/:id", handler.GetExportTask)
	router.POST("/api/v1/admin/traces/export-tasks/:id/cancel", handler.CancelExportTask)
	router.GET("/api/v1/admin/traces/export-tasks/:id/download", handler.DownloadExportTask)
	return router
}

func TestTraceHandlerListParsesFiltersAndPaginates(t *testing.T) {
	start := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	userID := int64(42)
	serviceStub := &traceCaptureServiceStub{
		listItems: []*service.ModelTraceCapture{{ID: 1, TaskID: "task-1"}},
		listResult: &pagination.PaginationResult{
			Total:    1,
			Page:     1,
			PageSize: 20,
			Pages:    1,
		},
	}
	handler := &TraceHandler{traceService: serviceStub}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces?model=%20gpt-4.1%20&user_id=42&start_date=2026-05-27&end_date=2026-05-27&timezone=UTC&keyword=%20incident%20", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "gpt-4.1", serviceStub.lastFilter.Model)
	require.Equal(t, &userID, serviceStub.lastFilter.UserID)
	require.NotNil(t, serviceStub.lastFilter.StartTime)
	require.NotNil(t, serviceStub.lastFilter.EndTime)
	require.True(t, serviceStub.lastFilter.StartTime.Equal(start))
	require.True(t, serviceStub.lastFilter.EndTime.Equal(end))
	require.Equal(t, "incident", serviceStub.lastFilter.Keyword)
}

func TestTraceHandlerBatchDeleteRejectsSystemPrincipal(t *testing.T) {
	handler := &TraceHandler{traceService: &traceCaptureServiceStub{}}
	router := setupTraceRouter(handler, "admin_api_key", &middleware.AuthSubject{
		PrincipalID:   "admin-key:test",
		PrincipalType: "admin_api_key",
		IsSystem:      true,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/traces/batch-delete", bytes.NewBufferString(`{"ids":[1,2]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTraceHandlerGetByIDNotFound(t *testing.T) {
	handler := &TraceHandler{traceService: &traceCaptureServiceStub{getErr: sql.ErrNoRows}}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTraceHandlerCreateRuleUsesRequestPayload(t *testing.T) {
	ruleSvc := &traceRuleServiceStub{
		createRule: &service.ModelTraceCaptureRule{ID: 3, Name: "capture-critical"},
	}
	handler := &TraceHandler{ruleService: ruleSvc}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	body := `{"name":"capture-critical","enabled":false,"priority":9,"model_patterns":["gpt-4.1"],"keywords":["incident"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/traces/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, ruleSvc.createInput)
	require.Equal(t, "capture-critical", ruleSvc.createInput.Name)
	require.False(t, ruleSvc.createInput.Enabled)
	require.Equal(t, []string{"gpt-4.1"}, ruleSvc.createInput.ModelPatterns)
}

func TestTraceHandlerUpdateRuleMergesExistingFields(t *testing.T) {
	minTokens := int64(100)
	ruleSvc := &traceRuleServiceStub{
		getRule: &service.ModelTraceCaptureRule{
			ID:            8,
			Name:          "existing",
			Enabled:       true,
			Priority:      1,
			ModelPatterns: []string{"gpt-4.1"},
			Keywords:      []string{"incident"},
			MinTokens:     &minTokens,
			SamplingRatio: 1,
		},
		updateRule: &service.ModelTraceCaptureRule{ID: 8, Name: "updated"},
	}
	handler := &TraceHandler{ruleService: ruleSvc}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/traces/rules/8", bytes.NewBufferString(`{"name":"updated","priority":7}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, ruleSvc.updateInput)
	require.Equal(t, int64(8), ruleSvc.updateInput.ID)
	require.Equal(t, "updated", ruleSvc.updateInput.Name)
	require.Equal(t, 7, ruleSvc.updateInput.Priority)
	require.Equal(t, []string{"gpt-4.1"}, ruleSvc.updateInput.ModelPatterns)
	require.Equal(t, []string{"incident"}, ruleSvc.updateInput.Keywords)
}

func TestTraceHandlerCreateExportTaskRequiresRootAdmin(t *testing.T) {
	handler := &TraceHandler{
		exportTaskService: &traceExportTaskServiceStub{},
		rootAdminLookup:   &rootAdminLookupStub{user: &service.User{ID: 1}},
	}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 2, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/traces/export-tasks", bytes.NewBufferString(`{"filters":{"model":"gpt-4.1"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTraceHandlerCreateExportTaskAndDownload(t *testing.T) {
	taskSvc := &traceExportTaskServiceStub{
		createTask: &service.TraceExportTask{ID: 5, Status: service.TraceExportTaskStatusPending},
		download: &service.TraceExportTaskDownload{
			Task:        &service.TraceExportTask{ID: 5},
			Reader:      io.NopCloser(bytes.NewBufferString(`[{"task_id":"task-1"}]`)),
			ContentType: "application/json; charset=utf-8",
			Filename:    "trace-export.json",
			Size:        int64(len(`[{"task_id":"task-1"}]`)),
		},
	}
	handler := &TraceHandler{
		exportTaskService: taskSvc,
		rootAdminLookup:   &rootAdminLookupStub{user: &service.User{ID: 1}},
	}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/traces/export-tasks", bytes.NewBufferString(`{"include_raw":true,"target_records":500,"filters":{"model":"gpt-4.1","keyword":"incident"}}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	require.Equal(t, http.StatusAccepted, createRec.Code)
	require.Equal(t, "gpt-4.1", taskSvc.lastCreateFilters.Model)
	require.Equal(t, "incident", taskSvc.lastCreateFilters.Keyword)
	require.True(t, taskSvc.lastIncludeRaw)
	require.Equal(t, int64(500), taskSvc.lastTargetRecords)
	require.Equal(t, int64(1), taskSvc.lastRequestedBy)

	downloadReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks/5/download", nil)
	downloadRec := httptest.NewRecorder()
	router.ServeHTTP(downloadRec, downloadReq)

	require.Equal(t, http.StatusOK, downloadRec.Code)
	require.Contains(t, downloadRec.Header().Get("Content-Disposition"), "trace-export.json")
	require.Equal(t, "5", downloadRec.Header().Get("X-Trace-Export-Task-ID"))
	require.Equal(t, `[{"task_id":"task-1"}]`, downloadRec.Body.String())
}

func TestTraceHandlerBatchDeleteNormalizesIDs(t *testing.T) {
	traceSvc := &traceCaptureServiceStub{}
	handler := &TraceHandler{traceService: traceSvc}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/traces/batch-delete", bytes.NewBufferString(`{"ids":[3,3,2,0,-1]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{3, 2}, traceSvc.deleteIDs)

	var payload response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
}

func TestTraceHandlerListRulesReturnsSuccessEnvelope(t *testing.T) {
	ruleSvc := &traceRuleServiceStub{
		listRules: []*service.ModelTraceCaptureRule{{
			ID:            17,
			Name:          "capture-incident",
			Enabled:       true,
			Priority:      10,
			ModelPatterns: []string{"gpt-5.4"},
			Keywords:      []string{"incident"},
			SamplingRatio: 1,
		}},
	}
	handler := &TraceHandler{ruleService: ruleSvc}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/rules", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeTraceHandlerResponseBody(t, rec)
	require.Equal(t, float64(0), body["code"])
	require.Equal(t, "success", body["message"])

	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(17), first["id"])
	require.Equal(t, "capture-incident", first["name"])
	require.Equal(t, []any{"gpt-5.4"}, first["model_patterns"])
}

func TestTraceHandlerGetRuleByIDReturnsSuccessEnvelope(t *testing.T) {
	ruleSvc := &traceRuleServiceStub{
		getRule: &service.ModelTraceCaptureRule{
			ID:            23,
			Name:          "capture-gpt",
			Enabled:       true,
			Priority:      6,
			ModelPatterns: []string{"gpt-*"},
			UserIDs:       []int64{42},
			Keywords:      []string{"alarm"},
			SamplingRatio: 1,
		},
	}
	handler := &TraceHandler{ruleService: ruleSvc}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/rules/23", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeTraceHandlerResponseBody(t, rec)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(23), data["id"])
	require.Equal(t, "capture-gpt", data["name"])
	require.Equal(t, []any{float64(42)}, data["user_ids"])
}

func TestTraceHandlerDeleteRuleReturnsSuccessEnvelope(t *testing.T) {
	ruleSvc := &traceRuleServiceStub{deleteOK: true}
	handler := &TraceHandler{ruleService: ruleSvc}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/traces/rules/19", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeTraceHandlerResponseBody(t, rec)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Trace rule deleted successfully", data["message"])
}

func TestTraceHandlerListExportTasksReturnsPaginatedEnvelope(t *testing.T) {
	taskSvc := &traceExportTaskServiceStub{
		listTasks: []service.TraceExportTask{{
			ID:          31,
			Status:      service.TraceExportTaskStatusRunning,
			Format:      service.TraceExportTaskFormatJSONArray,
			RequestedBy: 1,
		}},
		listResult: &pagination.PaginationResult{
			Total:    1,
			Page:     2,
			PageSize: 5,
			Pages:    1,
		},
	}
	handler := &TraceHandler{
		exportTaskService: taskSvc,
		rootAdminLookup:   &rootAdminLookupStub{user: &service.User{ID: 1}},
	}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks?page=2&page_size=5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeTraceHandlerResponseBody(t, rec)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), data["total"])
	require.Equal(t, float64(2), data["page"])
	require.Equal(t, float64(5), data["page_size"])

	items, ok := data["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(31), first["id"])
	require.Equal(t, service.TraceExportTaskStatusRunning, first["status"])
}

func TestTraceHandlerGetExportTaskReturnsSuccessEnvelope(t *testing.T) {
	taskSvc := &traceExportTaskServiceStub{
		getTask: &service.TraceExportTask{
			ID:          44,
			Status:      service.TraceExportTaskStatusSucceeded,
			Format:      service.TraceExportTaskFormatJSONArray,
			RequestedBy: 1,
		},
	}
	handler := &TraceHandler{
		exportTaskService: taskSvc,
		rootAdminLookup:   &rootAdminLookupStub{user: &service.User{ID: 1}},
	}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks/44", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeTraceHandlerResponseBody(t, rec)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(44), data["id"])
	require.Equal(t, service.TraceExportTaskStatusSucceeded, data["status"])
}

func TestTraceHandlerCancelExportTaskReturnsSuccessEnvelope(t *testing.T) {
	taskSvc := &traceExportTaskServiceStub{}
	handler := &TraceHandler{
		exportTaskService: taskSvc,
		rootAdminLookup:   &rootAdminLookupStub{user: &service.User{ID: 1}},
	}
	router := setupTraceRouter(handler, "jwt", &middleware.AuthSubject{UserID: 1, PrincipalType: "user"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/traces/export-tasks/51/cancel", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(1), taskSvc.lastCanceledBy)

	body := decodeTraceHandlerResponseBody(t, rec)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(51), data["id"])
	require.Equal(t, service.TraceExportTaskStatusCanceled, data["status"])
}

func decodeTraceHandlerResponseBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}
