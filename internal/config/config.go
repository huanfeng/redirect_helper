package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	
	"redirect_helper/pkg/utils"
)

type Config struct {
	Forwardings map[string]*ForwardingConfig `json:"forwardings"`
	Domains     map[string]*DomainConfig     `json:"domains"`
	Server      *ServerConfig                `json:"server"`
}

type ForwardingConfig struct {
	Name      string    `json:"name"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DomainConfig struct {
	Domain    string    `json:"domain"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ServerConfig struct {
	Port              string `json:"port"`
	AdminToken        string `json:"admin_token"`
	RedirectToken     string `json:"redirect_token"`
	DomainToken       string `json:"domain_token"`
	MaxRedirectCount  int    `json:"max_redirect_count"`
	MaxDomainCount    int    `json:"max_domain_count"`
}

func NewConfig() *Config {
	return &Config{
		Forwardings: make(map[string]*ForwardingConfig),
		Domains:     make(map[string]*DomainConfig),
		Server: &ServerConfig{
			Port:             "8001",
			AdminToken:       "",
			RedirectToken:    "",
			DomainToken:      "",
			MaxRedirectCount: 20,
			MaxDomainCount:   10,
		},
	}
}

var configPath string

func SetConfigPath(path string) {
	configPath = path
}

func GetConfigPath() string {
	if configPath != "" {
		return configPath
	}
	return "./redirect_helper.json"
}

// LoadConfig loads configuration for non-server modes (requires config to exist)
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s\nRun with -server flag to auto-create configuration", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	config := NewConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return config, nil
}

// LoadConfigForServer loads configuration for server mode (auto-creates and initializes tokens)
func LoadConfigForServer() (*Config, error) {
	configPath := GetConfigPath()
	
	// Ensure directory exists
	if err := ensureConfigDir(configPath); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create new config with auto-generated tokens
		config := NewConfig()
		
		// Generate tokens
		adminToken, err := utils.GenerateToken(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate admin token: %v", err)
		}
		
		redirectToken, err := utils.GenerateToken(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate redirect token: %v", err)
		}
		
		domainToken, err := utils.GenerateToken(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate domain token: %v", err)
		}
		
		// Set tokens
		config.Server.AdminToken = adminToken
		config.Server.RedirectToken = redirectToken
		config.Server.DomainToken = domainToken
		
		// Save config
		if err := config.Save(); err != nil {
			return nil, fmt.Errorf("failed to create config file: %v", err)
		}
		
		// Output generated tokens
		fmt.Printf("🎉 Configuration initialized successfully!\n")
		fmt.Printf("📁 Config file: %s\n", configPath)
		fmt.Printf("🔑 Generated tokens:\n")
		fmt.Printf("   Admin Token:    %s\n", adminToken)
		fmt.Printf("   Redirect Token: %s\n", redirectToken)
		fmt.Printf("   Domain Token:   %s\n", domainToken)
		fmt.Printf("💡 Save these tokens for API access!\n\n")
		
		return config, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	config := NewConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return config, nil
}

// ensureConfigDir ensures the directory for config file exists
func ensureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	if dir == "." {
		return nil // Current directory
	}
	
	return os.MkdirAll(dir, 0755)
}

func (c *Config) Save() error {
	configPath := GetConfigPath()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (c *Config) AddForwarding(name string) error {
	if _, exists := c.Forwardings[name]; exists {
		return fmt.Errorf("forwarding name already exists")
	}

	// Check max redirect count
	if len(c.Forwardings) >= c.Server.MaxRedirectCount {
		return fmt.Errorf("maximum redirect count (%d) reached", c.Server.MaxRedirectCount)
	}

	c.Forwardings[name] = &ForwardingConfig{
		Name:      name,
		Target:    "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return c.Save()
}

func (c *Config) SetTarget(name, token, target string) error {
	// Validate redirect token
	if c.Server.RedirectToken == "" || c.Server.RedirectToken != token {
		return fmt.Errorf("invalid redirect token")
	}

	// Create forwarding if it doesn't exist
	if _, exists := c.Forwardings[name]; !exists {
		if err := c.AddForwarding(name); err != nil {
			return err
		}
	}

	forwarding := c.Forwardings[name]
	forwarding.Target = target
	forwarding.UpdatedAt = time.Now()

	return c.Save()
}

func (c *Config) GetForwarding(name string) (*ForwardingConfig, error) {
	forwarding, exists := c.Forwardings[name]
	if !exists {
		return nil, fmt.Errorf("forwarding name not found")
	}

	return forwarding, nil
}

func (c *Config) GetTarget(name string) (string, error) {
	forwarding, err := c.GetForwarding(name)
	if err != nil {
		return "", err
	}

	if forwarding.Target == "" {
		return "", fmt.Errorf("target not set")
	}

	return forwarding.Target, nil
}

func (c *Config) ListForwardings() []*ForwardingConfig {
	result := make([]*ForwardingConfig, 0, len(c.Forwardings))
	for _, forwarding := range c.Forwardings {
		result = append(result, forwarding)
	}
	return result
}

func (c *Config) RemoveForwarding(name string) error {
	if _, exists := c.Forwardings[name]; !exists {
		return fmt.Errorf("forwarding name not found")
	}

	delete(c.Forwardings, name)
	return c.Save()
}

func (c *Config) UpdateTarget(name, target string) error {
	forwarding, exists := c.Forwardings[name]
	if !exists {
		return fmt.Errorf("forwarding name not found")
	}

	forwarding.Target = target
	forwarding.UpdatedAt = time.Now()

	return c.Save()
}

// Domain management methods
func (c *Config) AddDomain(domain string) error {
	if _, exists := c.Domains[domain]; exists {
		return fmt.Errorf("domain already exists")
	}

	// Check max domain count
	if len(c.Domains) >= c.Server.MaxDomainCount {
		return fmt.Errorf("maximum domain count (%d) reached", c.Server.MaxDomainCount)
	}

	c.Domains[domain] = &DomainConfig{
		Domain:    domain,
		Target:    "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return c.Save()
}

func (c *Config) SetDomainTarget(domain, token, target string) error {
	// Validate domain token
	if c.Server.DomainToken == "" || c.Server.DomainToken != token {
		return fmt.Errorf("invalid domain token")
	}

	// Create domain if it doesn't exist
	if _, exists := c.Domains[domain]; !exists {
		if err := c.AddDomain(domain); err != nil {
			return err
		}
	}

	domainConfig := c.Domains[domain]
	domainConfig.Target = target
	domainConfig.UpdatedAt = time.Now()

	return c.Save()
}

func (c *Config) GetDomain(domain string) (*DomainConfig, error) {
	domainConfig, exists := c.Domains[domain]
	if !exists {
		return nil, fmt.Errorf("domain not found")
	}

	return domainConfig, nil
}

func (c *Config) GetDomainTarget(domain string) (string, error) {
	domainConfig, err := c.GetDomain(domain)
	if err != nil {
		return "", err
	}

	if domainConfig.Target == "" {
		return "", fmt.Errorf("target not set")
	}

	return domainConfig.Target, nil
}

func (c *Config) ListDomains() []*DomainConfig {
	result := make([]*DomainConfig, 0, len(c.Domains))
	for _, domain := range c.Domains {
		result = append(result, domain)
	}
	return result
}

func (c *Config) RemoveDomain(domain string) error {
	if _, exists := c.Domains[domain]; !exists {
		return fmt.Errorf("domain not found")
	}

	delete(c.Domains, domain)
	return c.Save()
}

func (c *Config) UpdateDomainTarget(domain, target string) error {
	domainConfig, exists := c.Domains[domain]
	if !exists {
		return fmt.Errorf("domain not found")
	}

	domainConfig.Target = target
	domainConfig.UpdatedAt = time.Now()

	return c.Save()
}

// Admin token management
func (c *Config) SetAdminToken(token string) error {
	if c.Server == nil {
		c.Server = &ServerConfig{
			Port:             "8001",
			AdminToken:       "",
			RedirectToken:    "",
			DomainToken:      "",
			MaxRedirectCount: 20,
			MaxDomainCount:   10,
		}
	}
	c.Server.AdminToken = token
	return c.Save()
}

func (c *Config) SetRedirectToken(token string) error {
	if c.Server == nil {
		c.Server = &ServerConfig{
			Port:             "8001",
			AdminToken:       "",
			RedirectToken:    "",
			DomainToken:      "",
			MaxRedirectCount: 20,
			MaxDomainCount:   10,
		}
	}
	c.Server.RedirectToken = token
	return c.Save()
}

func (c *Config) SetDomainToken(token string) error {
	if c.Server == nil {
		c.Server = &ServerConfig{
			Port:             "8001",
			AdminToken:       "",
			RedirectToken:    "",
			DomainToken:      "",
			MaxRedirectCount: 20,
			MaxDomainCount:   10,
		}
	}
	c.Server.DomainToken = token
	return c.Save()
}

func (c *Config) GetAdminToken() string {
	if c.Server == nil {
		return ""
	}
	return c.Server.AdminToken
}

func (c *Config) ValidateAdminToken(token string) bool {
	adminToken := c.GetAdminToken()
	return adminToken != "" && adminToken == token
}

func (c *Config) GetRedirectToken() string {
	if c.Server == nil {
		return ""
	}
	return c.Server.RedirectToken
}

func (c *Config) GetDomainToken() string {
	if c.Server == nil {
		return ""
	}
	return c.Server.DomainToken
}

func (c *Config) ValidateRedirectToken(token string) bool {
	redirectToken := c.GetRedirectToken()
	return redirectToken != "" && redirectToken == token
}

func (c *Config) ValidateDomainToken(token string) bool {
	domainToken := c.GetDomainToken()
	return domainToken != "" && domainToken == token
}
