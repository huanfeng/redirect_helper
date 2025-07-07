package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"redirect_helper/internal/models"
	"redirect_helper/internal/storage"
	"redirect_helper/pkg/utils"
)

type Server struct {
	storage       storage.Storage
	domainStorage storage.DomainStorage
	configStorage *storage.ConfigStorage
	mux           *http.ServeMux
}

func NewServer(store interface{}) *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}

	if configStorage, ok := store.(*storage.ConfigStorage); ok {
		s.configStorage = configStorage
		s.storage = configStorage
		s.domainStorage = configStorage
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes - basic operations
	s.mux.HandleFunc("/api/list", s.handleListForwardings)
	s.mux.HandleFunc("/api/remove", s.handleRemoveForwarding)
	s.mux.HandleFunc("/api/update", s.handleUpdateSetTarget)

	// API routes - domain operations
	s.mux.HandleFunc("/api/list-domains", s.handleListDomains)
	s.mux.HandleFunc("/api/remove-domain", s.handleRemoveDomain)
	s.mux.HandleFunc("/api/update-domain", s.handleUpdateDomainTarget)

	// Legacy redirect route
	s.mux.HandleFunc("/go/", s.handleRedirect)

	// Catch-all handler for domain proxy (must be last)
	s.mux.HandleFunc("/", s.handleRequest)
}

func (s *Server) handleUpdateSetTarget(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	name := r.URL.Query().Get("name")
	token := r.URL.Query().Get("token")
	target := r.URL.Query().Get("target")

	if name == "" || token == "" || target == "" {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Missing required parameters: name, token, target",
		})
		return
	}

	if !s.isValidTarget(target) {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Invalid target format. Expected host:port",
		})
		return
	}

	err := s.storage.SetTarget(name, token, target)
	if err != nil {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.writeJSONResponse(w, http.StatusOK, models.Response{
		State: "success",
	})
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	// 从URL路径中提取名称，路径格式为 /go/name
	name := strings.TrimPrefix(r.URL.Path, "/go/")

	if name == "" {
		http.Error(w, "No forwarding name specified", http.StatusBadRequest)
		return
	}

	target, err := s.storage.GetTarget(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Forwarding error: %v", err), http.StatusNotFound)
		return
	}

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "http://" + target
	}

	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Redirect Helper</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 700px; margin: 0 auto; padding: 20px; }
        .api-section { margin: 20px 0; padding: 15px; background: #f5f5f5; border-radius: 5px; }
        code { background: #e8e8e8; padding: 2px 4px; border-radius: 3px; font-size: 12px; }
        .warning { color: #d63384; font-weight: bold; margin-top: 10px; }
        .method { color: #0066cc; font-weight: bold; }
    </style>
</head>
<body>
    <h1>🔄 Redirect Helper</h1>
    <p>Two redirect modes: path-based and domain-based</p>

    <div class="api-section">
        <h2>🔗 Path Redirects</h2>
        <p><strong>Access:</strong> <code>/go/&lt;name&gt;</code></p>
        <p><span class="method">GET</span> <strong>Update/Create:</strong> <code>/api/update?name=&lt;name&gt;&token=&lt;redirect_token&gt;&target=&lt;target&gt;</code></p>
        <p><span class="method">GET</span> <strong>List:</strong> <code>/api/list?admin_token=&lt;admin_token&gt;</code></p>
        <p><span class="method">DELETE</span> <strong>Remove:</strong> <code>/api/remove?name=&lt;name&gt;&admin_token=&lt;admin_token&gt;</code></p>
    </div>

    <div class="api-section">
        <h2>🌐 Domain Redirects</h2>
        <p><strong>Access:</strong> Direct domain access with full URL preservation</p>
        <p><span class="method">GET</span> <strong>Update/Create:</strong> <code>/api/update-domain?domain=&lt;domain&gt;&token=&lt;domain_token&gt;&target=&lt;target&gt;</code></p>
        <p><span class="method">GET</span> <strong>List:</strong> <code>/api/list-domains?admin_token=&lt;admin_token&gt;</code></p>
        <p><span class="method">DELETE</span> <strong>Remove:</strong> <code>/api/remove-domain?domain=&lt;domain&gt;&admin_token=&lt;admin_token&gt;</code></p>
    </div>

    <div class="warning">
        ⚠️ Management operations (list, remove) require admin token. Update operations require specific tokens.
    </div>
</body>
</html>
`
	w.Write([]byte(html))
}

func (s *Server) writeJSONResponse(w http.ResponseWriter, status int, response models.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) isValidTarget(target string) bool {
	if target == "" {
		return false
	}

	if strings.Contains(target, "://") {
		u, err := url.Parse(target)
		return err == nil && u.Host != ""
	}

	parts := strings.Split(target, ":")
	return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
}

func (s *Server) handleUpdateDomainTarget(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	domain := r.URL.Query().Get("domain")
	token := r.URL.Query().Get("token")
	target := r.URL.Query().Get("target")

	if domain == "" || token == "" || target == "" {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Missing required parameters: domain, token, target",
		})
		return
	}

	if !s.isValidTarget(target) {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Invalid target format. Expected URL or host:port",
		})
		return
	}

	if s.domainStorage == nil {
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: "Domain storage not available",
		})
		return
	}

	err := s.domainStorage.SetDomainTarget(domain, token, target)
	if err != nil {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.writeJSONResponse(w, http.StatusOK, models.Response{
		State: "success",
	})
}

func (s *Server) validateAdminToken(token string) bool {
	if s.configStorage == nil {
		return false
	}
	return s.configStorage.ValidateAdminToken(token)
}

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	// 检查管理员认证
	adminToken := r.URL.Query().Get("admin_token")
	if !s.validateAdminToken(adminToken) {
		s.writeJSONResponse(w, http.StatusUnauthorized, models.Response{
			State:   "error",
			Message: "Unauthorized access. Admin token required.",
		})
		return
	}

	if s.domainStorage == nil {
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: "Domain storage not available",
		})
		return
	}

	domains, err := s.domainStorage.ListDomains()
	if err != nil {
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	// 转换为公开信息，隐藏敏感token
	publicDomains := make([]*models.DomainEntryPublic, len(domains))
	for i, domain := range domains {
		publicDomains[i] = &models.DomainEntryPublic{
			Domain:    domain.Domain,
			Target:    domain.Target,
			CreatedAt: domain.CreatedAt,
			UpdatedAt: domain.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"state":   "success",
		"domains": publicDomains,
	})
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Check if this is a domain proxy request
	if s.domainStorage != nil {
		host := r.Host
		// Remove port from host if present
		if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}

		target, err := s.domainStorage.GetDomainTarget(host)
		if err == nil && target != "" {
			s.handleDomainProxy(w, r, target)
			return
		}
	}

	// If no domain match, handle as normal request
	if r.URL.Path == "/" {
		s.handleIndex(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handleDomainProxy(w http.ResponseWriter, r *http.Request, targetURL string) {
	target, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid target URL: %v", err), http.StatusInternalServerError)
		return
	}

	// 构建完整的目标URL，保持原始路径和查询参数
	target.Path = r.URL.Path
	target.RawQuery = r.URL.RawQuery
	target.Fragment = r.URL.Fragment

	// 使用HTTP重定向而不是反向代理
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// checkDomainRedirect 检查是否应该进行域名跳转
// 如果找到域名映射，执行跳转并返回 true
// 如果没有找到域名映射，返回 false 继续正常处理
func (s *Server) checkDomainRedirect(w http.ResponseWriter, r *http.Request) bool {
	if s.domainStorage == nil {
		return false
	}

	host := r.Host
	// Remove port from host if present
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	target, err := s.domainStorage.GetDomainTarget(host)
	if err != nil || target == "" {
		return false
	}

	// 找到域名映射，执行跳转
	s.handleDomainProxy(w, r, target)
	return true
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

// API handlers for forwarding management
func (s *Server) handleListForwardings(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	// 检查管理员认证
	adminToken := r.URL.Query().Get("admin_token")
	if !s.validateAdminToken(adminToken) {
		s.writeJSONResponse(w, http.StatusUnauthorized, models.Response{
			State:   "error",
			Message: "Unauthorized access. Admin token required.",
		})
		return
	}

	if s.storage == nil {
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: "Storage not available",
		})
		return
	}

	forwardings, err := s.storage.ListForwardings()
	if err != nil {
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"state":       "success",
		"forwardings": forwardings,
	})
}

func (s *Server) handleRemoveForwarding(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	if r.Method != http.MethodDelete {
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	// 检查管理员认证
	adminToken := r.URL.Query().Get("admin_token")
	if !s.validateAdminToken(adminToken) {
		s.writeJSONResponse(w, http.StatusUnauthorized, models.Response{
			State:   "error",
			Message: "Unauthorized access. Admin token required.",
		})
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Missing required parameter: name",
		})
		return
	}

	err := s.configStorage.RemoveForwarding(name)
	if err != nil {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.writeJSONResponse(w, http.StatusOK, models.Response{
		State: "success",
	})
}

func (s *Server) handleRemoveDomain(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	if r.Method != http.MethodDelete {
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	// 检查管理员认证
	adminToken := r.URL.Query().Get("admin_token")
	if !s.validateAdminToken(adminToken) {
		s.writeJSONResponse(w, http.StatusUnauthorized, models.Response{
			State:   "error",
			Message: "Unauthorized access. Admin token required.",
		})
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Missing required parameter: domain",
		})
		return
	}

	if s.domainStorage == nil {
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: "Domain storage not available",
		})
		return
	}

	err := s.configStorage.RemoveDomain(domain)
	if err != nil {
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.writeJSONResponse(w, http.StatusOK, models.Response{
		State: "success",
	})
}

// Helper function to generate tokens
func (s *Server) generateToken(length int) (string, error) {
	return utils.GenerateToken(length)
}
