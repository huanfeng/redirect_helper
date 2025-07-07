package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"redirect_helper/internal/config"
	"redirect_helper/internal/server"
	"redirect_helper/internal/storage"
	"redirect_helper/pkg/utils"
)

func main() {
	var (
		serverMode   = flag.Bool("server", false, "Run as server")
		port         = flag.String("port", "8001", "Server port")
		listMode     = flag.Bool("list", false, "List all forwarding entries")
		removeName   = flag.String("remove", "", "Remove a forwarding name")
		updateName   = flag.String("update", "", "Update/create target for a forwarding name")
		updateTarget = flag.String("target", "", "New target for update (use with -update)")
		configFile   = flag.String("config", "", "Configuration file path (default: ./redirect_helper.json)")

		// Domain management flags
		listDomains  = flag.Bool("list-domains", false, "List all domain mappings")
		removeDomain = flag.String("remove-domain", "", "Remove a domain mapping")
		updateDomain = flag.String("update-domain", "", "Update/create target for a domain mapping")

		// Token management flags
		resetAdminToken    = flag.Bool("reset-admin-token", false, "Reset admin token for API authentication")
		resetRedirectToken = flag.Bool("reset-redirect-token", false, "Reset redirect token for path redirects")
		resetDomainToken   = flag.Bool("reset-domain-token", false, "Reset domain token for domain redirects")
	)
	flag.Parse()

	if *configFile != "" {
		config.SetConfigPath(*configFile)
	}

	var cfg *config.Config
	var err error
	
	// Use different config loading based on server mode
	if *serverMode {
		cfg, err = config.LoadConfigForServer()
	} else {
		cfg, err = config.LoadConfig()
	}
	
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store := storage.NewConfigStorage(cfg)

	if *listMode {
		listForwardings(store)
		return
	}

	if *removeName != "" {
		removeForwarding(*removeName, store)
		return
	}

	if *updateName != "" {
		updateForwarding(*updateName, *updateTarget, store)
		return
	}

	// Domain management commands
	if *listDomains {
		listDomainMappings(store)
		return
	}

	if *removeDomain != "" {
		removeDomainMapping(*removeDomain, store)
		return
	}

	if *updateDomain != "" {
		updateDomainMapping(*updateDomain, *updateTarget, store)
		return
	}

	// Token management commands
	if *resetAdminToken {
		resetAdminTokenCmd(store)
		return
	}

	if *resetRedirectToken {
		resetRedirectTokenCmd(store)
		return
	}

	if *resetDomainToken {
		resetDomainTokenCmd(store)
		return
	}

	if *serverMode {
		startServer(*port, store, cfg)
		return
	}

	flag.Usage()
}


func listForwardings(store *storage.ConfigStorage) {
	forwardings, err := store.ListForwardings()
	if err != nil {
		log.Fatalf("Failed to list forwardings: %v", err)
	}

	if len(forwardings) == 0 {
		fmt.Println("No forwardings found")
		return
	}

	fmt.Println("Existing forwardings:")
	for _, f := range forwardings {
		fmt.Printf("Name: %s, Target: %s, Created: %s\n",
			f.Name, f.Target, f.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}

func removeForwarding(name string, store *storage.ConfigStorage) {
	err := store.RemoveForwarding(name)
	if err != nil {
		log.Fatalf("Failed to remove forwarding: %v", err)
	}

	fmt.Printf("Forwarding '%s' removed successfully\n", name)
}

func updateForwarding(name, target string, store *storage.ConfigStorage) {
	if target == "" {
		log.Fatal("Target is required for update. Use -target flag")
	}

	// 获取redirect token
	redirectToken := store.GetRedirectToken()
	if redirectToken == "" {
		log.Fatal("Redirect token not set. Use -reset-redirect-token to generate one")
	}

	err := store.SetTarget(name, redirectToken, target)
	if err != nil {
		log.Fatalf("Failed to update/create forwarding: %v", err)
	}

	fmt.Printf("Forwarding '%s' updated/created successfully with target: %s\n", name, target)
}

func startServer(port string, store *storage.ConfigStorage, cfg *config.Config) {
	// 使用配置文件中的端口，如果命令行没有指定非默认端口的话
	actualPort := port
	if port == "8001" && cfg.Server != nil && cfg.Server.Port != "" {
		actualPort = cfg.Server.Port
	}

	// 显示当前配置信息
	displayServerConfig(cfg, actualPort)

	srv := server.NewServer(store)
	fmt.Printf("🚀 Starting server on port %s...\n", actualPort)

	// 启动服务器（阻塞运行）
	fmt.Printf("Server started on port %s\n", actualPort)
	fmt.Printf("Press Ctrl+C to stop server\n\n")
	
	if err := srv.Start(":" + actualPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Domain management functions

func listDomainMappings(store *storage.ConfigStorage) {
	domains, err := store.ListDomains()
	if err != nil {
		log.Fatalf("Failed to list domain mappings: %v", err)
	}

	if len(domains) == 0 {
		fmt.Println("No domain mappings found")
		return
	}

	fmt.Println("Existing domain mappings:")
	for _, d := range domains {
		fmt.Printf("Domain: %s, Target: %s, Created: %s\n",
			d.Domain, d.Target, d.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}

func removeDomainMapping(domain string, store *storage.ConfigStorage) {
	err := store.RemoveDomain(domain)
	if err != nil {
		log.Fatalf("Failed to remove domain mapping: %v", err)
	}

	fmt.Printf("Domain mapping '%s' removed successfully\n", domain)
}

func updateDomainMapping(domain, target string, store *storage.ConfigStorage) {
	if target == "" {
		log.Fatal("Target is required for update. Use -target flag")
	}

	// 获取domain token
	domainToken := store.GetDomainToken()
	if domainToken == "" {
		log.Fatal("Domain token not set. Use -reset-domain-token to generate one")
	}

	err := store.SetDomainTarget(domain, domainToken, target)
	if err != nil {
		log.Fatalf("Failed to update/create domain mapping: %v", err)
	}

	fmt.Printf("Domain mapping '%s' updated/created successfully with target: %s\n", domain, target)
}

// Token management functions
func resetAdminTokenCmd(store *storage.ConfigStorage) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	err = store.SetAdminToken(token)
	if err != nil {
		log.Fatalf("Failed to set admin token: %v", err)
	}

	fmt.Printf("Admin token reset successfully\n")
	fmt.Printf("New admin token: %s\n", token)
}

func resetRedirectTokenCmd(store *storage.ConfigStorage) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	err = store.SetRedirectToken(token)
	if err != nil {
		log.Fatalf("Failed to set redirect token: %v", err)
	}

	fmt.Printf("Redirect token reset successfully\n")
	fmt.Printf("New redirect token: %s\n", token)
}

func resetDomainTokenCmd(store *storage.ConfigStorage) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	err = store.SetDomainToken(token)
	if err != nil {
		log.Fatalf("Failed to set domain token: %v", err)
	}

	fmt.Printf("Domain token reset successfully\n")
	fmt.Printf("New domain token: %s\n", token)
}


// displayServerConfig shows current configuration when starting server
func displayServerConfig(cfg *config.Config, port string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 Current Server Configuration")
	fmt.Println(strings.Repeat("=", 60))
	
	// Basic settings
	fmt.Printf("🌐 Server Port: %s\n", port)
	fmt.Printf("📁 Config File: %s\n", config.GetConfigPath())
	
	// Limits
	if cfg.Server != nil {
		fmt.Printf("📊 Limits: %d redirects, %d domains\n", 
			cfg.Server.MaxRedirectCount, cfg.Server.MaxDomainCount)
	}
	
	// Current entries count
	redirectCount := len(cfg.Forwardings)
	domainCount := len(cfg.Domains)
	fmt.Printf("📈 Current Usage: %d redirects, %d domains\n", redirectCount, domainCount)
	
	// Token status (without showing actual tokens)
	if cfg.Server != nil {
		adminSet := cfg.Server.AdminToken != ""
		redirectSet := cfg.Server.RedirectToken != ""
		domainSet := cfg.Server.DomainToken != ""
		
		fmt.Printf("🔑 Tokens Status:\n")
		fmt.Printf("   Admin Token:    %s\n", getTokenStatus(adminSet))
		fmt.Printf("   Redirect Token: %s\n", getTokenStatus(redirectSet))
		fmt.Printf("   Domain Token:   %s\n", getTokenStatus(domainSet))
	}
	
	// List existing entries if any
	if redirectCount > 0 {
		fmt.Printf("🔗 Active Redirects:\n")
		for name, forwarding := range cfg.Forwardings {
			target := forwarding.Target
			if target == "" {
				target = "[not configured]"
			}
			fmt.Printf("   %s → %s\n", name, target)
		}
	}
	
	if domainCount > 0 {
		fmt.Printf("🌐 Active Domains:\n")
		for domain, domainConfig := range cfg.Domains {
			target := domainConfig.Target
			if target == "" {
				target = "[not configured]"
			}
			fmt.Printf("   %s → %s\n", domain, target)
		}
	}
	
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

func getTokenStatus(isSet bool) string {
	if isSet {
		return "✅ Set"
	}
	return "❌ Not Set"
}
