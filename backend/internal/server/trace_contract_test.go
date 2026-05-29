//go:build unit

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTraceAdminContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	ruleID := int64(5)
	totalTokens := int64(160)
	capture := &service.ModelTraceCapture{
		ID:              11,
		TaskID:          "task-11",
		UserID:          traceContractInt64Ptr(42),
		APIKeyID:        traceContractInt64Ptr(7),
		CaptureRuleID:   &ruleID,
		Protocol:        "openai.responses",
		Model:           "gpt-5.4",
		TotalTokens:     &totalTokens,
		Scaffold:        "sub2api",
		ScaffoldVersion: "sub2api-taoding-trace-v1",
		Prompt:          json.RawMessage(`[{"role":"user","content":"incident 42"}]`),
		Candidates:      json.RawMessage(`[{"content":"incident resolved"}]`),
		Tools:           json.RawMessage(`[]`),
		Signature:       json.RawMessage(`{"available":false}`),
		Meta:            json.RawMessage(`{"source":"sub2api_gateway_capture"}`),
		RawRequestText:  `{"task_id":"task-11"}`,
		RawResponseText: `{"response":"incident resolved"}`,
		DedupeHash:      "dedupe-11",
		PromptHash:      "prompt-11",
		CreatedAt:       now,
	}
	rule := &service.ModelTraceCaptureRule{
		ID:            ruleID,
		Name:          "capture-incident",
		Enabled:       true,
		Priority:      100,
		ModelPatterns: []string{"gpt-5.4"},
		Keywords:      []string{"incident"},
		SamplingRatio: 1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	exportPath := filepath.Join(t.TempDir(), "trace-export.json")
	require.NoError(t, os.WriteFile(exportPath, []byte(`[{"task_id":"task-11","raw_request":{"task_id":"task-11"}}]`), 0o600))
	task := service.TraceExportTask{
		ID:               21,
		Status:           service.TraceExportTaskStatusSucceeded,
		Format:           service.TraceExportTaskFormatJSONArray,
		Filters:          service.TraceExportTaskFilters{Model: "gpt-5.4", CaptureRuleID: &ruleID, Keyword: "incident"},
		IncludeRaw:       true,
		RequestedBy:      1,
		DownloadFilename: "trace-export.json",
		FileSizeBytes:    58,
		TotalRecords:     1,
		ProcessedRecords: 1,
		CreatedAt:        now,
		UpdatedAt:        now,
		FilePath:         exportPath,
	}

	router := newTraceContractRouter(
		service.NewModelTraceCaptureService(&traceContractCaptureRepo{item: capture}),
		service.NewModelTraceCaptureRuleService(&traceContractRuleRepo{rule: rule}),
		service.NewTraceExportTaskService(&traceContractTaskRepo{task: task}),
		service.NewUserService(&traceContractUserRepo{
			firstAdmin: &service.User{ID: 1, Role: service.RoleAdmin, Status: service.StatusActive},
		}, nil, nil, nil),
	)

	t.Run("list and detail envelopes", func(t *testing.T) {
		listRec := httptest.NewRecorder()
		listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces?page=1&page_size=10&model=gpt-5.4&capture_rule_id=5", nil)
		router.ServeHTTP(listRec, listReq)
		require.Equal(t, http.StatusOK, listRec.Code)

		listBody := decodeTraceContractBody(t, listRec)
		require.Equal(t, float64(0), listBody["code"])
		listData := traceContractMap(t, listBody["data"])
		require.Equal(t, float64(1), listData["total"])
		require.Equal(t, float64(1), listData["page"])
		require.Equal(t, float64(10), listData["page_size"])
		items := traceContractSlice(t, listData["items"])
		require.Len(t, items, 1)
		item := traceContractMap(t, items[0])
		require.Equal(t, float64(11), item["id"])
		require.Equal(t, "task-11", item["task_id"])
		require.Equal(t, float64(5), item["capture_rule_id"])
		require.Equal(t, float64(160), item["total_tokens"])

		detailRec := httptest.NewRecorder()
		detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/11", nil)
		router.ServeHTTP(detailRec, detailReq)
		require.Equal(t, http.StatusOK, detailRec.Code)

		detailBody := decodeTraceContractBody(t, detailRec)
		detailData := traceContractMap(t, detailBody["data"])
		require.Equal(t, float64(11), detailData["id"])
		require.Equal(t, "openai.responses", detailData["protocol"])
		require.Equal(t, `{"task_id":"task-11"}`, detailData["raw_request_text"])
	})

	t.Run("rules and export task envelopes", func(t *testing.T) {
		rulesRec := httptest.NewRecorder()
		rulesReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/rules", nil)
		router.ServeHTTP(rulesRec, rulesReq)
		require.Equal(t, http.StatusOK, rulesRec.Code)

		rulesBody := decodeTraceContractBody(t, rulesRec)
		rulesData := traceContractSlice(t, rulesBody["data"])
		require.Len(t, rulesData, 1)
		firstRule := traceContractMap(t, rulesData[0])
		require.Equal(t, float64(5), firstRule["id"])
		require.Equal(t, "capture-incident", firstRule["name"])
		require.Equal(t, []any{"gpt-5.4"}, firstRule["model_patterns"])

		tasksRec := httptest.NewRecorder()
		tasksReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks?page=1&page_size=10", nil)
		router.ServeHTTP(tasksRec, tasksReq)
		require.Equal(t, http.StatusOK, tasksRec.Code)

		tasksBody := decodeTraceContractBody(t, tasksRec)
		tasksData := traceContractMap(t, tasksBody["data"])
		require.Equal(t, float64(1), tasksData["total"])
		taskItems := traceContractSlice(t, tasksData["items"])
		require.Len(t, taskItems, 1)
		firstTask := traceContractMap(t, taskItems[0])
		require.Equal(t, float64(21), firstTask["id"])
		require.Equal(t, "succeeded", firstTask["status"])
		require.Equal(t, "trace-export.json", firstTask["download_filename"])

		taskRec := httptest.NewRecorder()
		taskReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks/21", nil)
		router.ServeHTTP(taskRec, taskReq)
		require.Equal(t, http.StatusOK, taskRec.Code)

		taskBody := decodeTraceContractBody(t, taskRec)
		taskData := traceContractMap(t, taskBody["data"])
		require.Equal(t, float64(21), taskData["id"])
		require.Equal(t, float64(1), taskData["total_records"])
		require.Equal(t, float64(58), taskData["file_size_bytes"])
	})

	t.Run("download response headers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks/21/download", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
		require.Equal(t, "21", rec.Header().Get("X-Trace-Export-Task-ID"))
		require.Contains(t, rec.Header().Get("Content-Disposition"), `filename="trace-export.json"`)
		require.JSONEq(t, `[{"task_id":"task-11","raw_request":{"task_id":"task-11"}}]`, rec.Body.String())
	})
}

type traceContractCaptureRepo struct {
	item *service.ModelTraceCapture
}

func (r *traceContractCaptureRepo) Create(ctx context.Context, capture *service.ModelTraceCapture) (bool, error) {
	return false, nil
}

func (r *traceContractCaptureRepo) GetByID(ctx context.Context, id int64) (*service.ModelTraceCapture, error) {
	if r.item != nil && r.item.ID == id {
		return r.item, nil
	}
	return nil, nil
}

func (r *traceContractCaptureRepo) GetByTaskID(ctx context.Context, taskID string) (*service.ModelTraceCapture, error) {
	if r.item != nil && r.item.TaskID == taskID {
		return r.item, nil
	}
	return nil, nil
}

func (r *traceContractCaptureRepo) GetByMainSessionKey(ctx context.Context, mainSessionKey string) (*service.ModelTraceCapture, error) {
	if r.item != nil && r.item.MainSessionKey == mainSessionKey {
		return r.item, nil
	}
	return nil, nil
}

func (r *traceContractCaptureRepo) GetByDedupeHash(ctx context.Context, dedupeHash string) (*service.ModelTraceCapture, error) {
	if r.item != nil && r.item.DedupeHash == dedupeHash {
		return r.item, nil
	}
	return nil, nil
}

func (r *traceContractCaptureRepo) List(ctx context.Context, filter service.ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*service.ModelTraceCapture, *pagination.PaginationResult, error) {
	if r.item == nil {
		return nil, &pagination.PaginationResult{Total: 0, Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
	}
	return []*service.ModelTraceCapture{r.item}, &pagination.PaginationResult{Total: 1, Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (r *traceContractCaptureRepo) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*service.ModelTraceCapture, *pagination.PaginationResult, error) {
	return r.List(ctx, service.ModelTraceCaptureListFilter{}, params)
}

func (r *traceContractCaptureRepo) DeleteByID(ctx context.Context, id int64) (bool, error) {
	return false, nil
}

func (r *traceContractCaptureRepo) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	return 0, nil
}

type traceContractRuleRepo struct {
	rule *service.ModelTraceCaptureRule
}

func (r *traceContractRuleRepo) Create(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error) {
	return rule, nil
}

func (r *traceContractRuleRepo) Update(ctx context.Context, rule *service.ModelTraceCaptureRule) (*service.ModelTraceCaptureRule, error) {
	return rule, nil
}

func (r *traceContractRuleRepo) GetByID(ctx context.Context, id int64) (*service.ModelTraceCaptureRule, error) {
	if r.rule != nil && r.rule.ID == id {
		return r.rule, nil
	}
	return nil, nil
}

func (r *traceContractRuleRepo) List(ctx context.Context) ([]*service.ModelTraceCaptureRule, error) {
	if r.rule == nil {
		return nil, nil
	}
	return []*service.ModelTraceCaptureRule{r.rule}, nil
}

func (r *traceContractRuleRepo) DeleteByID(ctx context.Context, id int64) (bool, error) {
	return false, nil
}

type traceContractTaskRepo struct {
	task service.TraceExportTask
}

func (r *traceContractTaskRepo) Create(ctx context.Context, task *service.TraceExportTask) error {
	return nil
}

func (r *traceContractTaskRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.TraceExportTask, *pagination.PaginationResult, error) {
	return []service.TraceExportTask{r.task}, &pagination.PaginationResult{Total: 1, Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (r *traceContractTaskRepo) GetByID(ctx context.Context, id int64) (*service.TraceExportTask, error) {
	if r.task.ID == id {
		task := r.task
		return &task, nil
	}
	return nil, nil
}

func (r *traceContractTaskRepo) Cancel(ctx context.Context, id int64, canceledBy int64) (bool, error) {
	return false, nil
}

func (r *traceContractTaskRepo) ClaimNextPending(ctx context.Context, startedAt time.Time) (*service.TraceExportTask, error) {
	return nil, nil
}

func (r *traceContractTaskRepo) UpdateProgress(ctx context.Context, id int64, totalRecords, processedRecords int64, updatedAt time.Time) (bool, error) {
	return false, nil
}

func (r *traceContractTaskRepo) MarkSucceeded(ctx context.Context, id int64, filePath string, fileSizeBytes, totalRecords, processedRecords int64, finishedAt time.Time) (bool, error) {
	return false, nil
}

func (r *traceContractTaskRepo) MarkFailed(ctx context.Context, id int64, totalRecords, processedRecords int64, errorMessage string, finishedAt time.Time) (bool, error) {
	return false, nil
}

func (r *traceContractTaskRepo) MarkDownloaded(ctx context.Context, id int64, downloadedAt time.Time) (bool, error) {
	return false, nil
}

func (r *traceContractTaskRepo) FailStaleRunning(ctx context.Context, staleBefore time.Time, errorMessage string, failedAt time.Time) (int64, error) {
	return 0, nil
}

func (r *traceContractTaskRepo) ListReadyForFileCleanup(ctx context.Context, finishedBefore time.Time, limit int) ([]service.TraceExportTask, error) {
	return nil, nil
}

func (r *traceContractTaskRepo) ClearFileForTask(ctx context.Context, id int64, updatedAt time.Time) (bool, error) {
	return false, nil
}

type traceContractUserRepo struct {
	service.UserRepository
	firstAdmin *service.User
}

func (r *traceContractUserRepo) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	return r.firstAdmin, nil
}

func newTraceContractRouter(
	traceSvc *service.ModelTraceCaptureService,
	ruleSvc *service.ModelTraceCaptureRuleService,
	taskSvc *service.TraceExportTaskService,
	userSvc *service.UserService,
) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, PrincipalType: "user"})
		c.Set("auth_method", "jwt")
		c.Next()
	})

	handler := adminhandler.NewTraceHandler(traceSvc, ruleSvc, taskSvc, userSvc)
	router.GET("/api/v1/admin/traces", handler.List)
	router.GET("/api/v1/admin/traces/:id", handler.GetByID)
	router.GET("/api/v1/admin/traces/rules", handler.ListRules)
	router.GET("/api/v1/admin/traces/export-tasks", handler.ListExportTasks)
	router.GET("/api/v1/admin/traces/export-tasks/:id", handler.GetExportTask)
	router.GET("/api/v1/admin/traces/export-tasks/:id/download", handler.DownloadExportTask)
	return router
}

func decodeTraceContractBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func traceContractMap(t *testing.T, value any) map[string]any {
	t.Helper()

	out, ok := value.(map[string]any)
	require.True(t, ok)
	return out
}

func traceContractSlice(t *testing.T, value any) []any {
	t.Helper()

	out, ok := value.([]any)
	require.True(t, ok)
	return out
}

func traceContractInt64Ptr(value int64) *int64 {
	return &value
}
