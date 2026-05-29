package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ProxyHandler handles admin proxy management
type ProxyHandler struct {
	adminService service.AdminService
}

// NewProxyHandler creates a new admin proxy handler
func NewProxyHandler(adminService service.AdminService) *ProxyHandler {
	return &ProxyHandler{
		adminService: adminService,
	}
}

// CreateProxyRequest represents create proxy request
type CreateProxyRequest struct {
	Name     string `json:"name" binding:"required"`
	Protocol string `json:"protocol" binding:"required,oneof=http https socks5 socks5h"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateProxyRequest represents update proxy request
type UpdateProxyRequest struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol" binding:"omitempty,oneof=http https socks5 socks5h"`
	Host     string `json:"host"`
	Port     int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username *string `json:"username"`
	Password *string `json:"password"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive"`
}

const proxyAuthPairRequiredMessage = "proxy username and password must both be provided"

func validateProxyHost(host string) error {
	if service.NormalizeProxyHost(host) == "" {
		return infraerrors.BadRequest("PROXY_HOST_REQUIRED", "proxy host is required")
	}
	return nil
}

func validateProxyCredentials(username, password string) error {
	if service.HasPartialProxyAuth(username, password) {
		return infraerrors.BadRequest("PROXY_AUTH_PAIR_REQUIRED", proxyAuthPairRequiredMessage)
	}
	return nil
}

// List handles listing all proxies with pagination
// GET /api/v1/admin/proxies
func (h *ProxyHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	protocol := c.Query("protocol")
	status := c.Query("status")
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "id")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	// 标准化和验证 search 参数
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}

	proxies, total, err := h.adminService.ListProxiesWithAccountCount(c.Request.Context(), page, pageSize, protocol, status, search, sortBy, sortOrder)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.AdminProxyWithAccountCount, 0, len(proxies))
	for i := range proxies {
		out = append(out, *dto.ProxyWithAccountCountFromServiceAdmin(&proxies[i]))
	}
	response.Paginated(c, out, total, page, pageSize)
}

// GetAll handles getting all active proxies without pagination
// GET /api/v1/admin/proxies/all
// Optional query param: with_count=true to include account count per proxy
func (h *ProxyHandler) GetAll(c *gin.Context) {
	withCount := c.Query("with_count") == "true"

	if withCount {
		proxies, err := h.adminService.GetAllProxiesWithAccountCount(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		out := make([]dto.AdminProxyWithAccountCount, 0, len(proxies))
		for i := range proxies {
			out = append(out, *dto.ProxyWithAccountCountFromServiceAdmin(&proxies[i]))
		}
		response.Success(c, out)
		return
	}

	proxies, err := h.adminService.GetAllProxies(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.AdminProxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *dto.ProxyFromServiceAdmin(&proxies[i]))
	}
	response.Success(c, out)
}

// GetByID handles getting a proxy by ID
// GET /api/v1/admin/proxies/:id
func (h *ProxyHandler) GetByID(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	proxy, err := h.adminService.GetProxy(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.ProxyFromServiceAdmin(proxy))
}

// Create handles creating a new proxy
// POST /api/v1/admin/proxies
func (h *ProxyHandler) Create(c *gin.Context) {
	var req CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	executeAdminIdempotentJSON(c, "admin.proxies.create", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		host := service.NormalizeProxyHost(req.Host)
		if err := validateProxyHost(host); err != nil {
			return nil, err
		}
		if err := validateProxyCredentials(req.Username, req.Password); err != nil {
			return nil, err
		}
		proxy, err := h.adminService.CreateProxy(ctx, &service.CreateProxyInput{
			Name:     strings.TrimSpace(req.Name),
			Protocol: strings.TrimSpace(req.Protocol),
			Host:     host,
			Port:     req.Port,
			Username: req.Username,
			Password: req.Password,
		})
		if err != nil {
			return nil, err
		}
		return dto.ProxyFromServiceAdmin(proxy), nil
	})
}

// Update handles updating a proxy
// PUT /api/v1/admin/proxies/:id
func (h *ProxyHandler) Update(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	var req UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	existing, err := h.adminService.GetProxy(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	username := existing.Username
	if req.Username != nil {
		username = *req.Username
	}
	password := existing.Password
	if req.Password != nil {
		password = *req.Password
	}
	if err := validateProxyCredentials(username, password); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	host := strings.TrimSpace(req.Host)
	if host != "" {
		host = service.NormalizeProxyHost(host)
		if err := validateProxyHost(host); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	updateInput := &service.UpdateProxyInput{
		Name:     strings.TrimSpace(req.Name),
		Protocol: strings.TrimSpace(req.Protocol),
		Host:     host,
		Port:     req.Port,
		Status:   strings.TrimSpace(req.Status),
	}
	if req.Username != nil {
		updateInput.Username = *req.Username
	}
	if req.Password != nil {
		updateInput.Password = *req.Password
	}

	proxy, err := h.adminService.UpdateProxy(c.Request.Context(), proxyID, updateInput)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.ProxyFromServiceAdmin(proxy))
}

// Delete handles deleting a proxy
// DELETE /api/v1/admin/proxies/:id
func (h *ProxyHandler) Delete(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	err = h.adminService.DeleteProxy(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Proxy deleted successfully"})
}

// BatchDelete handles batch deleting proxies
// POST /api/v1/admin/proxies/batch-delete
func (h *ProxyHandler) BatchDelete(c *gin.Context) {
	type BatchDeleteRequest struct {
		IDs []int64 `json:"ids" binding:"required,min=1"`
	}

	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.adminService.BatchDeleteProxies(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// Test handles testing proxy connectivity
// POST /api/v1/admin/proxies/:id/test
func (h *ProxyHandler) Test(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	result, err := h.adminService.TestProxy(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// CheckQuality handles checking proxy quality across common AI targets.
// POST /api/v1/admin/proxies/:id/quality-check
func (h *ProxyHandler) CheckQuality(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	result, err := h.adminService.CheckProxyQuality(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// GetStats handles getting proxy statistics
// GET /api/v1/admin/proxies/:id/stats
func (h *ProxyHandler) GetStats(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	if _, err := h.adminService.GetProxy(c.Request.Context(), proxyID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.ErrorWithDetails(c, http.StatusNotImplemented, "proxy request stats unavailable", "PROXY_STATS_UNAVAILABLE", map[string]string{
		"proxy_id": strconv.FormatInt(proxyID, 10),
	})
}

// GetProxyAccounts handles getting accounts using a proxy
// GET /api/v1/admin/proxies/:id/accounts
func (h *ProxyHandler) GetProxyAccounts(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	accounts, err := h.adminService.GetProxyAccounts(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.ProxyAccountSummary, 0, len(accounts))
	for i := range accounts {
		out = append(out, *dto.ProxyAccountSummaryFromService(&accounts[i]))
	}
	response.Success(c, out)
}

// BatchCreateProxyItem represents a single proxy in batch create request
type BatchCreateProxyItem struct {
	Protocol string `json:"protocol" binding:"required,oneof=http https socks5 socks5h"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// BatchCreateRequest represents batch create proxies request
type BatchCreateRequest struct {
	Proxies []BatchCreateProxyItem `json:"proxies" binding:"required,min=1"`
}

// BatchCreate handles batch creating proxies
// POST /api/v1/admin/proxies/batch
func (h *ProxyHandler) BatchCreate(c *gin.Context) {
	var req BatchCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	existingProxies, err := h.listProxiesFiltered(c.Request.Context(), "", "", "", "id", "desc")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	existingKeys := make(map[string]struct{}, len(existingProxies))
	for i := range existingProxies {
		existingKeys[service.ProxyIdentityKey(
			existingProxies[i].Protocol,
			existingProxies[i].Host,
			existingProxies[i].Port,
			existingProxies[i].Username,
			existingProxies[i].Password,
		)] = struct{}{}
	}

	created := 0
	skipped := 0

	for _, item := range req.Proxies {
		host := service.NormalizeProxyHost(item.Host)
		protocol := strings.TrimSpace(item.Protocol)
		username := item.Username
		password := item.Password

		if err := validateProxyHost(host); err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if err := validateProxyCredentials(username, password); err != nil {
			response.ErrorFrom(c, err)
			return
		}

		key := service.ProxyIdentityKey(protocol, host, item.Port, username, password)
		if _, exists := existingKeys[key]; exists {
			skipped++
			continue
		}

		_, err = h.adminService.CreateProxy(c.Request.Context(), &service.CreateProxyInput{
			Name:     "default",
			Protocol: protocol,
			Host:     host,
			Port:     item.Port,
			Username: username,
			Password: password,
		})
		if err != nil {
			skipped++
			continue
		}

		existingKeys[key] = struct{}{}
		created++
	}

	response.Success(c, gin.H{
		"created": created,
		"skipped": skipped,
	})
}
