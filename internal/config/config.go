package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Forwardings map[string]*ForwardingConfig `json:"forwardings"`
	Domains     map[string]*DomainConfig     `json:"domains"`
	Server      *ServerConfig                `json:"server"`
}

type ForwardingConfig struct {
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DomainConfig struct {
	Domain    string    `json:"domain"`
	Token     string    `json:"token"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ServerConfig struct {
	Port       string `json:"port"`
	AdminToken string `json:"admin_token"`
}

func NewConfig() *Config {
	return &Config{
		Forwardings: make(map[string]*ForwardingConfig),
		Domains:     make(map[string]*DomainConfig),
		Server: &ServerConfig{
			Port:       "8001",
			AdminToken: "",
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

func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := NewConfig()
		if err := config.Save(); err != nil {
			return nil, fmt.Errorf("failed to create config file: %v", err)
		}
		return config, nil
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

func (c *Config) AddForwarding(name, token string) error {
	if _, exists := c.Forwardings[name]; exists {
		return fmt.Errorf("forwarding name already exists")
	}

	c.Forwardings[name] = &ForwardingConfig{
		Name:      name,
		Token:     token,
		Target:    "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return c.Save()
}

func (c *Config) SetTarget(name, token, target string) error {
	forwarding, exists := c.Forwardings[name]
	if !exists {
		return fmt.Errorf("forwarding name not found")
	}

	if forwarding.Token != token {
		return fmt.Errorf("invalid token")
	}

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
func (c *Config) AddDomain(domain, token string) error {
	if _, exists := c.Domains[domain]; exists {
		return fmt.Errorf("domain already exists")
	}

	c.Domains[domain] = &DomainConfig{
		Domain:    domain,
		Token:     token,
		Target:    "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return c.Save()
}

func (c *Config) SetDomainTarget(domain, token, target string) error {
	domainConfig, exists := c.Domains[domain]
	if !exists {
		return fmt.Errorf("domain not found")
	}

	if domainConfig.Token != token {
		return fmt.Errorf("invalid token")
	}

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
			Port:       "8001",
			AdminToken: "",
		}
	}
	c.Server.AdminToken = token
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
