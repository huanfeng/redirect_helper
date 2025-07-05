package main

import (
	"flag"
	"fmt"
	"log"

	"redirect_helper/internal/config"
	"redirect_helper/internal/server"
	"redirect_helper/internal/storage"
	"redirect_helper/pkg/utils"
)

func main() {
	var (
		createName   = flag.String("create", "", "Create a new forwarding name")
		serverMode   = flag.Bool("server", false, "Run as server")
		port         = flag.String("port", "8001", "Server port")
		listMode     = flag.Bool("list", false, "List all forwarding entries")
		removeName   = flag.String("remove", "", "Remove a forwarding name")
		updateName   = flag.String("update", "", "Update target for a forwarding name")
		updateTarget = flag.String("target", "", "New target for update (use with -update)")
		configFile   = flag.String("config", "", "Configuration file path (default: ./redirect_helper.json)")

		// Domain management flags
		createDomain = flag.String("create-domain", "", "Create a new domain mapping")
		listDomains  = flag.Bool("list-domains", false, "List all domain mappings")
		removeDomain = flag.String("remove-domain", "", "Remove a domain mapping")
		updateDomain = flag.String("update-domain", "", "Update target for a domain mapping")
	)
	flag.Parse()

	if *configFile != "" {
		config.SetConfigPath(*configFile)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store := storage.NewConfigStorage(cfg)

	if *createName != "" {
		createForwarding(*createName, store)
		return
	}

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
	if *createDomain != "" {
		createDomainMapping(*createDomain, store)
		return
	}

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

	if *serverMode {
		startServer(*port, store)
		return
	}

	flag.Usage()
}

func createForwarding(name string, store *storage.ConfigStorage) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	err = store.CreateForwarding(name, token)
	if err != nil {
		log.Fatalf("Failed to create forwarding: %v", err)
	}

	fmt.Printf("Forwarding created successfully:\n")
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Token: %s\n", token)
	fmt.Printf("Use this token to set the target via API\n")
	fmt.Printf("Config saved to: %s\n", config.GetConfigPath())
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

	err := store.UpdateTarget(name, target)
	if err != nil {
		log.Fatalf("Failed to update forwarding: %v", err)
	}

	fmt.Printf("Forwarding '%s' updated successfully with target: %s\n", name, target)
}

func startServer(port string, store *storage.ConfigStorage) {
	srv := server.NewServer(store)
	fmt.Printf("Starting server on port %s...\n", port)
	if err := srv.Start(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Domain management functions
func createDomainMapping(domain string, store *storage.ConfigStorage) {
	token, err := utils.GenerateToken(32)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	err = store.CreateDomain(domain, token)
	if err != nil {
		log.Fatalf("Failed to create domain mapping: %v", err)
	}

	fmt.Printf("Domain mapping created successfully:\n")
	fmt.Printf("Domain: %s\n", domain)
	fmt.Printf("Token: %s\n", token)
	fmt.Printf("Use this token to set the target via API\n")
	fmt.Printf("Config saved to: %s\n", config.GetConfigPath())
}

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

	err := store.UpdateDomainTarget(domain, target)
	if err != nil {
		log.Fatalf("Failed to update domain mapping: %v", err)
	}

	fmt.Printf("Domain mapping '%s' updated successfully with target: %s\n", domain, target)
}
