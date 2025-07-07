package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

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

	name := r.URL.Query().Get("name")
	token := r.URL.Query().Get("token")
	target := r.URL.Query().Get("target")

	params := map[string]string{
		"name":   name,
		"token":  token,
		"target": target,
	}

	if r.Method != http.MethodGet {
		s.logAPIRequest(r, "/api/update", params, "method_not_allowed", http.StatusMethodNotAllowed)
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	if name == "" || token == "" || target == "" {
		s.logAPIRequest(r, "/api/update", params, "missing_parameters", http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Missing required parameters: name, token, target",
		})
		return
	}

	if !s.isValidTarget(target) {
		s.logAPIRequest(r, "/api/update", params, "invalid_target_format", http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Invalid target format. Expected host:port",
		})
		return
	}

	err := s.storage.SetTarget(name, token, target)
	if err != nil {
		s.logAPIRequest(r, "/api/update", params, fmt.Sprintf("error:%s", err.Error()), http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.logAPIRequest(r, "/api/update", params, "success", http.StatusOK)
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
        .admin-section { margin: 20px 0; padding: 15px; background: #e8f4f8; border-radius: 5px; border-left: 4px solid #0066cc; }
        code { background: #e8e8e8; padding: 2px 4px; border-radius: 3px; font-size: 12px; }
        .warning { color: #d63384; font-weight: bold; margin-top: 10px; }
        .method { color: #0066cc; font-weight: bold; }
        .form-group { margin: 10px 0; }
        .form-group label { display: block; margin-bottom: 5px; font-weight: bold; }
        .form-group input { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
        .btn { padding: 10px 15px; margin: 5px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
        .btn-primary { background: #0066cc; color: white; }
        .btn-primary:hover { background: #0052a3; }
        .btn-secondary { background: #6c757d; color: white; }
        .btn-secondary:hover { background: #5a6268; }
        .result { margin-top: 15px; padding: 10px; border-radius: 4px; display: none; }
        .result.success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .result.error { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
        .entries { margin-top: 10px; }
        .entry { margin: 5px 0; padding: 8px; background: #f8f9fa; border-radius: 4px; }
        .entry strong { color: #0066cc; }
        .entry-time { color: #6c757d; font-size: 12px; }
    </style>
</head>
<body>
    <h1>🔄 Redirect Helper</h1>
    <p>Two redirect modes: path-based and domain-based</p>

    <div class="admin-section">
        <h2>🔑 Admin Management</h2>
        <div class="form-group">
            <label for="adminToken">Admin Token:</label>
            <input type="text" id="adminToken" placeholder="Enter your admin token">
        </div>
        <button class="btn btn-primary" onclick="listRedirects()">List Redirects</button>
        <button class="btn btn-secondary" onclick="listDomains()">List Domains</button>
        <div id="result" class="result"></div>
    </div>

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

    <script>
        function showResult(message, isSuccess) {
            const result = document.getElementById('result');
            result.className = 'result ' + (isSuccess ? 'success' : 'error');
            result.innerHTML = message;
            result.style.display = 'block';
        }

        function formatDate(dateString) {
            return new Date(dateString).toLocaleString();
        }

        function listRedirects() {
            const adminToken = document.getElementById('adminToken').value;
            if (!adminToken) {
                showResult('Please enter admin token', false);
                return;
            }

            fetch('/api/list?admin_token=' + encodeURIComponent(adminToken))
                .then(response => response.json())
                .then(data => {
                    if (data.state === 'success') {
                        let html = '<h3>📋 Path Redirects:</h3>';
                        if (data.forwardings && data.forwardings.length > 0) {
                            html += '<div class="entries">';
                            data.forwardings.forEach(forwarding => {
                                html += '<div class="entry">';
                                html += '<strong>' + forwarding.name + '</strong> → ' + forwarding.target;
                                html += '<div class="entry-time">Created: ' + formatDate(forwarding.created_at) + '</div>';
                                html += '</div>';
                            });
                            html += '</div>';
                        } else {
                            html += '<p>No redirects found</p>';
                        }
                        showResult(html, true);
                    } else {
                        showResult('Error: ' + data.message, false);
                    }
                })
                .catch(error => {
                    showResult('Request failed: ' + error.message, false);
                });
        }

        function listDomains() {
            const adminToken = document.getElementById('adminToken').value;
            if (!adminToken) {
                showResult('Please enter admin token', false);
                return;
            }

            fetch('/api/list-domains?admin_token=' + encodeURIComponent(adminToken))
                .then(response => response.json())
                .then(data => {
                    if (data.state === 'success') {
                        let html = '<h3>🌐 Domain Redirects:</h3>';
                        if (data.domains && data.domains.length > 0) {
                            html += '<div class="entries">';
                            data.domains.forEach(domain => {
                                html += '<div class="entry">';
                                html += '<strong>' + domain.domain + '</strong> → ' + domain.target;
                                html += '<div class="entry-time">Created: ' + formatDate(domain.created_at) + '</div>';
                                html += '</div>';
                            });
                            html += '</div>';
                        } else {
                            html += '<p>No domain redirects found</p>';
                        }
                        showResult(html, true);
                    } else {
                        showResult('Error: ' + data.message, false);
                    }
                })
                .catch(error => {
                    showResult('Request failed: ' + error.message, false);
                });
        }
    </script>
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

	domain := r.URL.Query().Get("domain")
	token := r.URL.Query().Get("token")
	target := r.URL.Query().Get("target")

	params := map[string]string{
		"domain": domain,
		"token":  token,
		"target": target,
	}

	if r.Method != http.MethodGet {
		s.logAPIRequest(r, "/api/update-domain", params, "method_not_allowed", http.StatusMethodNotAllowed)
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	if domain == "" || token == "" || target == "" {
		s.logAPIRequest(r, "/api/update-domain", params, "missing_parameters", http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Missing required parameters: domain, token, target",
		})
		return
	}

	if !s.isValidTarget(target) {
		s.logAPIRequest(r, "/api/update-domain", params, "invalid_target_format", http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Invalid target format. Expected URL or host:port",
		})
		return
	}

	if s.domainStorage == nil {
		s.logAPIRequest(r, "/api/update-domain", params, "domain_storage_unavailable", http.StatusInternalServerError)
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: "Domain storage not available",
		})
		return
	}

	err := s.domainStorage.SetDomainTarget(domain, token, target)
	if err != nil {
		s.logAPIRequest(r, "/api/update-domain", params, fmt.Sprintf("error:%s", err.Error()), http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.logAPIRequest(r, "/api/update-domain", params, "success", http.StatusOK)
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

// logAPIRequest logs API requests with relevant information
func (s *Server) logAPIRequest(r *http.Request, endpoint string, params map[string]string, result string, status int) {
	clientIP := r.RemoteAddr
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = forwardedFor
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// Format parameters for logging (hide sensitive tokens)
	logParams := make(map[string]string)
	for k, v := range params {
		if strings.Contains(strings.ToLower(k), "token") && v != "" {
			logParams[k] = v[:8] + "..."  // Show only first 8 chars
		} else {
			logParams[k] = v
		}
	}
	
	log.Printf("[API] %s | %s %s | %s | Status: %d | Params: %v | Result: %s", 
		timestamp, r.Method, endpoint, clientIP, status, logParams, result)
}

// API handlers for forwarding management
func (s *Server) handleListForwardings(w http.ResponseWriter, r *http.Request) {
	// 先检查是否为域名跳转
	if s.checkDomainRedirect(w, r) {
		return
	}

	adminToken := r.URL.Query().Get("admin_token")
	params := map[string]string{
		"admin_token": adminToken,
	}

	if r.Method != http.MethodGet {
		s.logAPIRequest(r, "/api/list", params, "method_not_allowed", http.StatusMethodNotAllowed)
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	// 检查管理员认证
	if !s.validateAdminToken(adminToken) {
		s.logAPIRequest(r, "/api/list", params, "unauthorized", http.StatusUnauthorized)
		s.writeJSONResponse(w, http.StatusUnauthorized, models.Response{
			State:   "error",
			Message: "Unauthorized access. Admin token required.",
		})
		return
	}

	if s.storage == nil {
		s.logAPIRequest(r, "/api/list", params, "storage_unavailable", http.StatusInternalServerError)
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: "Storage not available",
		})
		return
	}

	forwardings, err := s.storage.ListForwardings()
	if err != nil {
		s.logAPIRequest(r, "/api/list", params, fmt.Sprintf("error:%s", err.Error()), http.StatusInternalServerError)
		s.writeJSONResponse(w, http.StatusInternalServerError, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.logAPIRequest(r, "/api/list", params, fmt.Sprintf("success:%d_items", len(forwardings)), http.StatusOK)
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

	adminToken := r.URL.Query().Get("admin_token")
	name := r.URL.Query().Get("name")
	params := map[string]string{
		"admin_token": adminToken,
		"name":        name,
	}

	if r.Method != http.MethodDelete {
		s.logAPIRequest(r, "/api/remove", params, "method_not_allowed", http.StatusMethodNotAllowed)
		s.writeJSONResponse(w, http.StatusMethodNotAllowed, models.Response{
			State:   "error",
			Message: "Method not allowed",
		})
		return
	}

	// 检查管理员认证
	if !s.validateAdminToken(adminToken) {
		s.logAPIRequest(r, "/api/remove", params, "unauthorized", http.StatusUnauthorized)
		s.writeJSONResponse(w, http.StatusUnauthorized, models.Response{
			State:   "error",
			Message: "Unauthorized access. Admin token required.",
		})
		return
	}

	if name == "" {
		s.logAPIRequest(r, "/api/remove", params, "missing_name_parameter", http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: "Missing required parameter: name",
		})
		return
	}

	err := s.configStorage.RemoveForwarding(name)
	if err != nil {
		s.logAPIRequest(r, "/api/remove", params, fmt.Sprintf("error:%s", err.Error()), http.StatusBadRequest)
		s.writeJSONResponse(w, http.StatusBadRequest, models.Response{
			State:   "error",
			Message: err.Error(),
		})
		return
	}

	s.logAPIRequest(r, "/api/remove", params, "success", http.StatusOK)
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
