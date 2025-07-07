package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
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

	srv := server.NewServer(store)
	fmt.Printf("Starting server on port %s...\n", actualPort)

	// 在后台启动服务器
	go func() {
		if err := srv.Start(":" + actualPort); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// 等待服务器启动
	fmt.Printf("Server started on port %s\n", actualPort)
	fmt.Printf("Press Enter to access settings menu...\n")

	// 交互式菜单
	runInteractiveMenu(store, cfg)
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

// Interactive menu system
func runInteractiveMenu(store *storage.ConfigStorage, cfg *config.Config) {
	scanner := bufio.NewScanner(os.Stdin)

	// 等待用户按回车键进入菜单
	scanner.Scan()

	for {
		// 显示主菜单
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("🔄 Redirect Helper - Interactive Menu")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("1. Settings")
		fmt.Println("2. Forwardings")
		fmt.Println("3. Domains")
		fmt.Println("q. Quit")
		fmt.Print("Select option: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			settingsMenu(store, cfg, scanner)
		case "2":
			forwardingsMenu(store, scanner)
		case "3":
			domainsMenu(store, scanner)
		case "q", "Q":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}

func settingsMenu(store *storage.ConfigStorage, cfg *config.Config, scanner *bufio.Scanner) {
	for {
		fmt.Println("\n" + strings.Repeat("-", 40))
		fmt.Println("⚙️  Settings")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println("1. View")
		fmt.Println("2. Change port")
		fmt.Println("3. Reset admin token")
		fmt.Println("4. Reset redirect token")
		fmt.Println("5. Reset domain token")
		fmt.Println("b. Back")
		fmt.Print("Select option: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			viewSettings(cfg, store)
		case "2":
			changePort(store, cfg, scanner)
		case "3":
			generateAdminToken(store, scanner)
		case "4":
			generateRedirectToken(store, scanner)
		case "5":
			generateDomainToken(store, scanner)
		case "b", "B":
			fmt.Println("") // 添加一个空行，然后直接返回到主菜单
			return
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}

func forwardingsMenu(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	for {
		fmt.Println("\n" + strings.Repeat("-", 40))
		fmt.Println("🔗 Forwardings")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println("1. List")
		fmt.Println("2. Update/Create")
		fmt.Println("3. Remove")
		fmt.Println("b. Back")
		fmt.Print("Select option: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			listForwardingsInteractive(store)
		case "2":
			updateForwardingInteractive(store, scanner)
		case "3":
			removeForwardingInteractive(store, scanner)
		case "b", "B":
			fmt.Println("") // 添加一个空行，然后直接返回到主菜单
			return
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}

func domainsMenu(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	for {
		fmt.Println("\n" + strings.Repeat("-", 40))
		fmt.Println("🌐 Domains")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println("1. List")
		fmt.Println("2. Update/Create")
		fmt.Println("3. Remove")
		fmt.Println("b. Back")
		fmt.Print("Select option: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			listDomainsInteractive(store)
		case "2":
			updateDomainInteractive(store, scanner)
		case "3":
			removeDomainInteractive(store, scanner)
		case "b", "B":
			fmt.Println("") // 添加一个空行，然后直接返回到主菜单
			return
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}

// Settings menu functions
func viewSettings(cfg *config.Config, store *storage.ConfigStorage) {
	fmt.Println("\n📋 Current Settings:")
	if cfg.Server != nil {
		fmt.Printf("Port: %s\n", cfg.Server.Port)
		fmt.Printf("Max Redirects: %d\n", cfg.Server.MaxRedirectCount)
		fmt.Printf("Max Domains: %d\n", cfg.Server.MaxDomainCount)
		
		if cfg.Server.AdminToken != "" {
			fmt.Printf("Admin Token: %s\n", cfg.Server.AdminToken)
		} else {
			fmt.Println("Admin Token: Not set")
		}
		
		if cfg.Server.RedirectToken != "" {
			fmt.Printf("Redirect Token: %s\n", cfg.Server.RedirectToken)
		} else {
			fmt.Println("Redirect Token: Not set")
		}
		
		if cfg.Server.DomainToken != "" {
			fmt.Printf("Domain Token: %s\n", cfg.Server.DomainToken)
		} else {
			fmt.Println("Domain Token: Not set")
		}
	}
	fmt.Printf("Config file: %s\n", config.GetConfigPath())
}

func changePort(store *storage.ConfigStorage, cfg *config.Config, scanner *bufio.Scanner) {
	fmt.Printf("Current port: %s\n", cfg.Server.Port)
	fmt.Print("Enter new port: ")

	if !scanner.Scan() {
		return
	}

	newPort := strings.TrimSpace(scanner.Text())
	if newPort == "" {
		fmt.Println("Port cannot be empty")
		return
	}

	// 验证端口号
	if port, err := strconv.Atoi(newPort); err != nil || port < 1 || port > 65535 {
		fmt.Println("Invalid port number. Must be between 1 and 65535")
		return
	}

	cfg.Server.Port = newPort
	if err := cfg.Save(); err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		return
	}

	fmt.Printf("Port changed to %s. Restart server to apply changes.\n", newPort)
}

func generateAdminToken(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		fmt.Printf("Failed to generate token: %v\n", err)
		return
	}

	if err := store.SetAdminToken(token); err != nil {
		fmt.Printf("Failed to set admin token: %v\n", err)
		return
	}

	fmt.Printf("New admin token generated: %s\n", token)
}

func generateRedirectToken(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		fmt.Printf("Failed to generate token: %v\n", err)
		return
	}

	if err := store.SetRedirectToken(token); err != nil {
		fmt.Printf("Failed to set redirect token: %v\n", err)
		return
	}

	fmt.Printf("New redirect token generated: %s\n", token)
}

func generateDomainToken(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		fmt.Printf("Failed to generate token: %v\n", err)
		return
	}

	if err := store.SetDomainToken(token); err != nil {
		fmt.Printf("Failed to set domain token: %v\n", err)
		return
	}

	fmt.Printf("New domain token generated: %s\n", token)
}

// Forwardings menu functions
func listForwardingsInteractive(store *storage.ConfigStorage) {
	forwardings, err := store.ListForwardings()
	if err != nil {
		fmt.Printf("Failed to list forwardings: %v\n", err)
		return
	}

	if len(forwardings) == 0 {
		fmt.Println("No forwardings found")
		return
	}

	fmt.Println("\n📋 Current Forwardings:")
	for i, f := range forwardings {
		fmt.Printf("%d. Name: %s, Target: %s, Created: %s\n",
			i+1, f.Name, f.Target, f.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}


func updateForwardingInteractive(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	// Check if redirect token is set
	redirectToken := store.GetRedirectToken()
	if redirectToken == "" {
		fmt.Println("❌ Redirect token not set. Please generate one in Settings menu first.")
		return
	}

	fmt.Print("Enter forwarding name: ")
	if !scanner.Scan() {
		return
	}

	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		fmt.Println("Name cannot be empty")
		return
	}

	fmt.Print("Enter target: ")
	if !scanner.Scan() {
		return
	}

	target := strings.TrimSpace(scanner.Text())
	if target == "" {
		fmt.Println("Target cannot be empty")
		return
	}

	if err := store.SetTarget(name, redirectToken, target); err != nil {
		fmt.Printf("Failed to update/create forwarding: %v\n", err)
		return
	}

	fmt.Printf("Forwarding '%s' updated/created successfully with target: %s\n", name, target)
}

func removeForwardingInteractive(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	fmt.Print("Enter forwarding name to remove: ")
	if !scanner.Scan() {
		return
	}

	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		fmt.Println("Name cannot be empty")
		return
	}

	if err := store.RemoveForwarding(name); err != nil {
		fmt.Printf("Failed to remove forwarding: %v\n", err)
		return
	}

	fmt.Printf("Forwarding '%s' removed successfully\n", name)
}

// Domains menu functions
func listDomainsInteractive(store *storage.ConfigStorage) {
	domains, err := store.ListDomains()
	if err != nil {
		fmt.Printf("Failed to list domains: %v\n", err)
		return
	}

	if len(domains) == 0 {
		fmt.Println("No domains found")
		return
	}

	fmt.Println("\n📋 Current Domains:")
	for i, d := range domains {
		fmt.Printf("%d. Domain: %s, Target: %s, Created: %s\n",
			i+1, d.Domain, d.Target, d.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}


func updateDomainInteractive(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	// Check if domain token is set
	domainToken := store.GetDomainToken()
	if domainToken == "" {
		fmt.Println("❌ Domain token not set. Please generate one in Settings menu first.")
		return
	}

	fmt.Print("Enter domain name: ")
	if !scanner.Scan() {
		return
	}

	domain := strings.TrimSpace(scanner.Text())
	if domain == "" {
		fmt.Println("Domain cannot be empty")
		return
	}

	fmt.Print("Enter target: ")
	if !scanner.Scan() {
		return
	}

	target := strings.TrimSpace(scanner.Text())
	if target == "" {
		fmt.Println("Target cannot be empty")
		return
	}

	if err := store.SetDomainTarget(domain, domainToken, target); err != nil {
		fmt.Printf("Failed to update/create domain: %v\n", err)
		return
	}

	fmt.Printf("Domain '%s' updated/created successfully with target: %s\n", domain, target)
}

func removeDomainInteractive(store *storage.ConfigStorage, scanner *bufio.Scanner) {
	fmt.Print("Enter domain name to remove: ")
	if !scanner.Scan() {
		return
	}

	domain := strings.TrimSpace(scanner.Text())
	if domain == "" {
		fmt.Println("Domain cannot be empty")
		return
	}

	if err := store.RemoveDomain(domain); err != nil {
		fmt.Printf("Failed to remove domain: %v\n", err)
		return
	}

	fmt.Printf("Domain '%s' removed successfully\n", domain)
}
