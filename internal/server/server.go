package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"redirect_helper/internal/models"
	"redirect_helper/internal/storage"
)

type Server struct {
	storage       storage.Storage
	domainStorage storage.DomainStorage
	mux           *http.ServeMux
}

func NewServer(store interface{}) *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}

	if storage, ok := store.(storage.Storage); ok {
		s.storage = storage
	}

	if domainStorage, ok := store.(storage.DomainStorage); ok {
		s.domainStorage = domainStorage
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/set", s.handleSetTarget)
	s.mux.HandleFunc("/api/set-domain", s.handleSetDomainTarget)
	s.mux.HandleFunc("/api/list-domains", s.handleListDomains)

	// Legacy redirect route
	s.mux.HandleFunc("/go/", s.handleRedirect)

	// Catch-all handler for domain proxy (must be last)
	s.mux.HandleFunc("/", s.handleRequest)
}

func (s *Server) handleSetTarget(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "text/html")
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Redirect Helper</title>
</head>
<body>
    <h1>Redirect Helper Service</h1>
    <p>Use the following format to access forwarding:</p>
    <p><code>/go/&lt;name&gt;</code> - Redirect to configured target</p>
    <p><code>/api/set?name=&lt;name&gt;&token=&lt;token&gt;&target=&lt;target&gt;</code> - Set target for forwarding</p>
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

func (s *Server) handleSetDomainTarget(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"state":   "success",
		"domains": domains,
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
