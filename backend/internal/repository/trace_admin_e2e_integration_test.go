//go:build integration

package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTraceAdminClosedLoopFromGatewayCaptureToExportDownload(t *testing.T) {
	ctx := context.Background()
	sqlTx := testTx(t)

	captureRepo := &modelTraceCaptureRepository{sql: sqlTx}
	ruleRepo := &modelTraceCaptureRuleRepository{sql: sqlTx}
	taskRepo := &traceExportTaskRepository{sql: sqlTx}

	traceSvc := service.NewModelTraceCaptureService(captureRepo)
	ruleSvc := service.NewModelTraceCaptureRuleService(ruleRepo)
	exportTaskSvc := service.NewTraceExportTaskService(taskRepo)

	rootAdminID := int64(7001)
	userSvc := service.NewUserService(&traceAdminE2EUserRepoStub{
		firstAdmin: &service.User{ID: rootAdminID, Role: service.RoleAdmin, Status: service.StatusActive},
	}, nil, nil, nil)

	minTokens := int64(100)
	rule, err := ruleSvc.Create(ctx, &service.ModelTraceCaptureRule{
		Name:          "capture-incident-e2e",
		Enabled:       true,
		Priority:      100,
		ModelPatterns: []string{"gpt-5.4"},
		Keywords:      []string{"incident"},
		MinTokens:     &minTokens,
		SamplingRatio: 1,
	})
	require.NoError(t, err)
	require.NotZero(t, rule.ID)

	userID := int64(4201)
	apiKeyID := int64(5301)
	groupID := int64(6401)
	accountID := int64(7501)
	now := time.Now().UTC()

	requestBody := `{
		"task_id":"task-trace-e2e",
		"model":"gpt-5.4",
		"instructions":"Root incident workflow",
		"input":[
			{
				"role":"user",
				"content":[
					{"type":"input_text","text":"Investigate incident 42 and capture the full trace"}
				]
			}
		],
		"tools":[
			{
				"type":"function",
				"function":{
					"name":"lookup_incident",
					"description":"Lookup incident details",
					"parameters":{"type":"object","properties":{"id":{"type":"string"}}}
				}
			}
		]
	}`
	responseBody := `{
		"type":"response.completed",
		"response":{
			"id":"resp_trace_e2e",
			"model":"gpt-5.4",
			"usage":{"input_tokens":128,"output_tokens":32,"total_tokens":160},
			"output":[
				{
					"type":"message",
					"role":"assistant",
					"content":[
						{"type":"output_text","text":"incident resolved"}
					]
				}
			]
		}
	}`

	traceRecorder := service.NewModelInteractionTraceService(nil, captureRepo, ruleRepo)

	created, err := traceRecorder.RecordGatewayTraceCaptures(ctx, []service.GatewayTraceCaptureEntry{
		{
			Stage:       service.GatewayTraceStageClientRequest,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			Body:        requestBody,
			CapturedAt:  now.Format(time.RFC3339Nano),
			Meta: map[string]any{
				"account_id": accountID,
			},
		},
		{
			Stage:       service.GatewayTraceStageUpstreamRequest,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			Body:        requestBody,
			CapturedAt:  now.Add(100 * time.Millisecond).Format(time.RFC3339Nano),
			Meta: map[string]any{
				"account_id": accountID,
			},
		},
		{
			Stage:       service.GatewayTraceStageUpstreamResponse,
			Protocol:    "openai.responses",
			ContentType: "application/json",
			StatusCode:  http.StatusOK,
			Body:        responseBody,
			CapturedAt:  now.Add(200 * time.Millisecond).Format(time.RFC3339Nano),
			Meta: map[string]any{
				"account_id":          accountID,
				"upstream_request_id": "req_upstream_trace_e2e",
			},
		},
	}, service.GatewayTraceRecordInput{
		UserID:    &userID,
		APIKeyID:  &apiKeyID,
		GroupID:   &groupID,
		RequestID: "req_gateway_trace_e2e",
		Method:    http.MethodPost,
		Path:      "/openai/v1/responses",
	})
	require.NoError(t, err)
	require.True(t, created)

	capture, err := traceSvc.GetByTaskID(ctx, "task-trace-e2e")
	require.NoError(t, err)
	require.NotNil(t, capture)
	require.NotZero(t, capture.ID)
	require.NotNil(t, capture.CaptureRuleID)
	require.Equal(t, rule.ID, *capture.CaptureRuleID)
	require.Equal(t, "gpt-5.4", capture.Model)
	require.NotNil(t, capture.TotalTokens)
	require.Equal(t, int64(160), *capture.TotalTokens)
	require.Contains(t, string(capture.Prompt), "incident 42")
	require.Contains(t, string(capture.RawResponse), "incident resolved")

	handler := adminhandler.NewTraceHandler(traceSvc, ruleSvc, exportTaskSvc, userSvc)
	router := newTraceAdminE2ERouter(handler, middleware.AuthSubject{
		UserID:        rootAdminID,
		PrincipalType: "user",
	})

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/traces?model=gpt-5.4&keyword=incident&capture_rule_id="+jsonNumber(rule.ID),
		nil,
	)
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)

	var listPayload response.Response
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listPayload))
	listData := mustMap(t, listPayload.Data)
	require.Equal(t, float64(1), listData["total"])
	items := mustSlice(t, listData["items"])
	require.Len(t, items, 1)
	firstTrace := mustMap(t, items[0])
	require.Equal(t, float64(capture.ID), firstTrace["id"])
	require.Equal(t, "task-trace-e2e", firstTrace["task_id"])
	require.Equal(t, "gpt-5.4", firstTrace["model"])
	require.Equal(t, float64(rule.ID), firstTrace["capture_rule_id"])

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/"+jsonNumber(capture.ID), nil)
	router.ServeHTTP(detailRec, detailReq)
	require.Equal(t, http.StatusOK, detailRec.Code)

	var detailPayload response.Response
	require.NoError(t, json.Unmarshal(detailRec.Body.Bytes(), &detailPayload))
	traceDetail := mustMap(t, detailPayload.Data)
	require.Equal(t, float64(capture.ID), traceDetail["id"])
	require.Equal(t, "task-trace-e2e", traceDetail["task_id"])
	require.Equal(t, "openai.responses", traceDetail["protocol"])
	require.Equal(t, float64(160), traceDetail["total_tokens"])

	createExportRec := httptest.NewRecorder()
	createExportReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/traces/export-tasks", bytes.NewBufferString(`{
		"include_raw": true,
		"filters": {
			"model": "gpt-5.4",
			"keyword": "incident",
			"capture_rule_id": `+jsonNumber(rule.ID)+`
		}
	}`))
	createExportReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createExportRec, createExportReq)
	require.Equal(t, http.StatusAccepted, createExportRec.Code)

	var createExportPayload response.Response
	require.NoError(t, json.Unmarshal(createExportRec.Body.Bytes(), &createExportPayload))
	createdTask := mustMap(t, createExportPayload.Data)
	taskID := int64(createdTask["id"].(float64))
	require.NotZero(t, taskID)
	require.Equal(t, service.TraceExportTaskStatusPending, createdTask["status"])

	listTasksRec := httptest.NewRecorder()
	listTasksReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks?page=1&page_size=10", nil)
	router.ServeHTTP(listTasksRec, listTasksReq)
	require.Equal(t, http.StatusOK, listTasksRec.Code)

	var listTasksPayload response.Response
	require.NoError(t, json.Unmarshal(listTasksRec.Body.Bytes(), &listTasksPayload))
	listTasksData := mustMap(t, listTasksPayload.Data)
	taskItems := mustSlice(t, listTasksData["items"])
	require.Len(t, taskItems, 1)
	require.Equal(t, float64(taskID), mustMap(t, taskItems[0])["id"])

	exportBytes, err := json.Marshal([]*service.ModelTraceCaptureExport{
		capture.Export(service.ModelTraceCaptureExportOptions{IncludeRaw: true}),
	})
	require.NoError(t, err)

	exportPath := filepath.Join(t.TempDir(), "trace-export-e2e.json")
	require.NoError(t, os.WriteFile(exportPath, exportBytes, 0o600))
	claimed, err := taskRepo.ClaimNextPending(ctx, time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, taskID, claimed.ID)
	markedSucceeded, err := taskRepo.MarkSucceeded(ctx, taskID, exportPath, int64(len(exportBytes)), 1, 1, time.Now().UTC())
	require.NoError(t, err)
	require.True(t, markedSucceeded)

	getTaskRec := httptest.NewRecorder()
	getTaskReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks/"+jsonNumber(taskID), nil)
	router.ServeHTTP(getTaskRec, getTaskReq)
	require.Equal(t, http.StatusOK, getTaskRec.Code)

	var getTaskPayload response.Response
	require.NoError(t, json.Unmarshal(getTaskRec.Body.Bytes(), &getTaskPayload))
	taskDetail := mustMap(t, getTaskPayload.Data)
	require.Equal(t, float64(taskID), taskDetail["id"])
	require.Equal(t, service.TraceExportTaskStatusSucceeded, taskDetail["status"])
	require.Equal(t, float64(1), taskDetail["total_records"])
	require.Equal(t, float64(len(exportBytes)), taskDetail["file_size_bytes"])

	downloadRec := httptest.NewRecorder()
	downloadReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/traces/export-tasks/"+jsonNumber(taskID)+"/download", nil)
	router.ServeHTTP(downloadRec, downloadReq)
	require.Equal(t, http.StatusOK, downloadRec.Code)
	require.Equal(t, "application/json; charset=utf-8", downloadRec.Header().Get("Content-Type"))
	require.Equal(t, jsonNumber(taskID), downloadRec.Header().Get("X-Trace-Export-Task-ID"))
	require.Contains(t, downloadRec.Header().Get("Content-Disposition"), ".json")
	require.JSONEq(t, string(exportBytes), downloadRec.Body.String())
	require.Contains(t, downloadRec.Body.String(), `"task_id":"task-trace-e2e"`)
	require.Contains(t, downloadRec.Body.String(), `"raw_request"`)
}

type traceAdminE2EUserRepoStub struct {
	service.UserRepository
	firstAdmin *service.User
}

func (s *traceAdminE2EUserRepoStub) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	return s.firstAdmin, nil
}

func newTraceAdminE2ERouter(handler *adminhandler.TraceHandler, subject middleware.AuthSubject) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), subject)
		c.Set("auth_method", "jwt")
		c.Next()
	})

	router.GET("/api/v1/admin/traces", handler.List)
	router.GET("/api/v1/admin/traces/:id", handler.GetByID)
	router.GET("/api/v1/admin/traces/export-tasks", handler.ListExportTasks)
	router.POST("/api/v1/admin/traces/export-tasks", handler.CreateExportTask)
	router.GET("/api/v1/admin/traces/export-tasks/:id", handler.GetExportTask)
	router.GET("/api/v1/admin/traces/export-tasks/:id/download", handler.DownloadExportTask)
	return router
}

func jsonNumber(value int64) string {
	return strconv.FormatInt(value, 10)
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	require.True(t, ok)
	return out
}

func mustSlice(t *testing.T, value any) []any {
	t.Helper()
	out, ok := value.([]any)
	require.True(t, ok)
	return out
}
