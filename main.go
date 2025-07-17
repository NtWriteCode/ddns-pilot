package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type DDNSPilot struct {
	config *AppConfig
	ddns   *DDNSManager
}

func main() {
	// Parse command line flags
	var (
		webMode     = flag.Bool("web", false, "Start web interface")
		cliMode     = flag.Bool("cli", false, "Run in CLI mode")
		updateAll   = flag.Bool("update", false, "Update all enabled DNS records")
		addRecord   = flag.Bool("add", false, "Add a new DNS record interactively")
		listRecords = flag.Bool("list", false, "List all configured DNS records")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *showHelp {
		showUsage()
		return
	}

	// Load configuration
	if err := ensureConfigDir(); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create DDNS Pilot instance
	pilot := &DDNSPilot{
		config: config,
		ddns:   NewDDNSManager(config),
	}

	// Determine mode
	if *webMode {
		pilot.startWebMode()
	} else if *cliMode || *updateAll || *addRecord || *listRecords {
		pilot.runCLIMode(*updateAll, *addRecord, *listRecords)
	} else {
		// Default: start web mode if no arguments provided
		fmt.Println("Starting DDNS Pilot in web mode. Use --help for CLI options.")
		pilot.startWebMode()
	}
}

func (p *DDNSPilot) startWebMode() {
	// Start session cleanup routine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sessionManager.CleanupExpiredSessions()
				rateLimiter.CleanupOldAttempts()
			}
		}
	}()

	// Start auto-update routine if enabled
	if p.config.AutoUpdate {
		go p.startAutoUpdateRoutine()
	}

	// Setup HTTP routes
	http.HandleFunc("/login", p.handleLogin)
	http.HandleFunc("/change-password", p.handleChangePassword)
	http.HandleFunc("/logout", sessionAuth(p.handleLogout, p.config))
	http.HandleFunc("/", sessionAuth(p.handleIndex, p.config))
	http.HandleFunc("/add-record", sessionAuth(p.handleAddRecord, p.config))
	http.HandleFunc("/edit-record", sessionAuth(p.handleEditRecord, p.config))
	http.HandleFunc("/remove-record", sessionAuth(p.handleRemoveRecord, p.config))
	http.HandleFunc("/toggle-record", sessionAuth(p.handleToggleRecord, p.config))
	http.HandleFunc("/update-records", sessionAuth(p.handleUpdateRecords, p.config))
	http.HandleFunc("/update-single", sessionAuth(p.handleUpdateSingle, p.config))
	http.HandleFunc("/settings", sessionAuth(p.handleSettings, p.config))
	http.HandleFunc("/api/stats", sessionAuth(p.handleStatsAPI, p.config))
	http.HandleFunc("/api", sessionAuth(p.handleAPI, p.config))

	// Determine port to use
	port := strconv.Itoa(p.config.Web.Port)
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("Starting DDNS Pilot on port %s", port)
	log.Printf("Access the web interface at: http://localhost:%s", port)
	log.Printf("Username: admin (password is securely hashed)")

	if len(p.config.Records) == 0 {
		log.Printf("No DNS records configured. Add some via the web interface.")
	} else {
		log.Printf("Managing %d DNS record(s)", len(p.config.Records))
	}

	if p.config.AutoUpdate {
		log.Printf("Auto-update enabled (every %d minutes)", p.config.UpdateInterval)
	} else {
		log.Printf("Auto-update disabled - manual updates only")
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down...")
		os.Exit(0)
	}()

	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func (p *DDNSPilot) runCLIMode(updateAll, addRecord, listRecords bool) {
	switch {
	case updateAll:
		p.cliUpdateAll()
	case addRecord:
		p.cliAddRecord()
	case listRecords:
		p.cliListRecords()
	default:
		showUsage()
	}
}

func (p *DDNSPilot) cliUpdateAll() {
	fmt.Println("🔄 Updating all enabled DNS records...")

	if len(p.config.Records) == 0 {
		fmt.Println("❌ No DNS records configured.")
		return
	}

	results := p.ddns.UpdateAllRecords()

	fmt.Printf("\n📊 Update Results:\n")
	for _, result := range results {
		if result.Success {
			fmt.Printf("✅ %s: %s\n", result.RecordName, result.Message)
			if result.OldIP != result.NewIP && result.NewIP != "" {
				fmt.Printf("   %s → %s\n", result.OldIP, result.NewIP)
			}
		} else {
			fmt.Printf("❌ %s: %s\n", result.RecordName, result.Message)
		}
	}
}

func (p *DDNSPilot) cliAddRecord() {
	fmt.Println("🔧 Add a new DNS record")

	var record DDNSRecord

	// Get record name
	fmt.Print("Full record name (e.g. home.example.com): ")
	fmt.Scanln(&record.RecordName)
	record.RecordName = strings.TrimSpace(record.RecordName)

	if record.RecordName == "" {
		fmt.Println("❌ Record name cannot be empty")
		return
	}

	// Check if record already exists
	if _, err := p.config.GetRecord(record.RecordName); err == nil {
		fmt.Println("❌ Record already exists")
		return
	}

	// Get API token
	fmt.Print("CloudFlare API Token: ")
	fmt.Scanln(&record.APIToken)
	record.APIToken = strings.TrimSpace(record.APIToken)

	if record.APIToken == "" {
		fmt.Println("❌ API token cannot be empty")
		return
	}

	// Get proxied setting
	var proxiedInput string
	fmt.Print("Proxied via CloudFlare? (y/n): ")
	fmt.Scanln(&proxiedInput)
	record.Proxied = strings.ToLower(strings.TrimSpace(proxiedInput)) == "y"

	// Get notes
	fmt.Print("Notes (optional): ")
	fmt.Scanln(&record.Notes)

	// Try to auto-fill zone and record IDs
	fmt.Println("🔍 Looking up zone and record information...")

	zoneName, err := p.ddns.ExtractZoneName(record.RecordName)
	if err != nil {
		fmt.Printf("❌ Invalid record name: %v\n", err)
		return
	}

	zoneID, err := p.ddns.GetZoneID(record.APIToken, zoneName)
	if err != nil {
		fmt.Printf("❌ Failed to get zone ID: %v\n", err)
		return
	}
	record.ZoneID = zoneID

	recordID, err := p.ddns.GetRecordID(record.APIToken, record.ZoneID, record.RecordName)
	if err != nil {
		fmt.Printf("❌ Failed to get record ID: %v\n", err)
		return
	}
	record.RecordID = recordID

	// Add the record
	p.config.AddRecord(record)

	if err := p.config.save(); err != nil {
		fmt.Printf("❌ Failed to save config: %v\n", err)
		return
	}

	fmt.Println("✅ DNS record added successfully!")
}

func (p *DDNSPilot) cliListRecords() {
	fmt.Println("📋 Configured DNS Records:")

	if len(p.config.Records) == 0 {
		fmt.Println("No records configured.")
		return
	}

	for i, record := range p.config.Records {
		status := "✅ Enabled"
		if !record.Enabled {
			status = "❌ Disabled"
		}

		fmt.Printf("\n%d. %s\n", i+1, record.RecordName)
		fmt.Printf("   Status: %s\n", status)
		fmt.Printf("   Proxied: %v\n", record.Proxied)
		fmt.Printf("   Last IP: %s\n", record.LastIP)
		fmt.Printf("   Last Updated: %s\n", record.LastUpdated)
		if record.Notes != "" {
			fmt.Printf("   Notes: %s\n", record.Notes)
		}
	}
}

func (p *DDNSPilot) startAutoUpdateRoutine() {
	ticker := time.NewTicker(time.Duration(p.config.UpdateInterval) * time.Minute)
	defer ticker.Stop()

	log.Printf("Auto-update routine started (interval: %d minutes)", p.config.UpdateInterval)

	for {
		select {
		case <-ticker.C:
			log.Println("Running auto-update...")
			results := p.ddns.UpdateAllRecords()

			// Log results
			for _, result := range results {
				if result.Success {
					if result.OldIP != result.NewIP && result.NewIP != "" {
						log.Printf("Auto-update: %s updated %s → %s", result.RecordName, result.OldIP, result.NewIP)
					}
				} else {
					log.Printf("Auto-update error: %s - %s", result.RecordName, result.Message)
				}
			}
		}
	}
}

func showUsage() {
	fmt.Println("🚁 DDNS Pilot - CloudFlare Dynamic DNS Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ddns-pilot [OPTIONS]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --web         Start web interface (default)")
	fmt.Println("  --cli         Run in CLI mode")
	fmt.Println("  --update      Update all enabled DNS records")
	fmt.Println("  --add         Add a new DNS record interactively")
	fmt.Println("  --list        List all configured DNS records")
	fmt.Println("  --help        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ddns-pilot --web                 # Start web interface")
	fmt.Println("  ddns-pilot --update              # Update all records")
	fmt.Println("  ddns-pilot --add                 # Add new record")
	fmt.Println("  ddns-pilot --list                # List records")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  PORT                             # Override web interface port")
	fmt.Println()
}
