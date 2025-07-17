package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// sessionAuth checks for valid session authentication
func sessionAuth(next http.HandlerFunc, config *AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login page and login POST
		if r.URL.Path == "/login" {
			next(w, r)
			return
		}

		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			// No session cookie, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate session
		_, valid := sessionManager.GetSession(cookie.Value)
		if !valid {
			// Invalid session, redirect to login
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    "",
				Expires:  time.Unix(0, 0),
				HttpOnly: true,
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

func (p *DDNSPilot) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Get client IP for rate limiting
	clientIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}

	if r.Method == "GET" {
		// Check if already logged in
		if cookie, err := r.Cookie("session_id"); err == nil {
			if _, valid := sessionManager.GetSession(cookie.Value); valid {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		// Check if IP is blocked
		isBlocked := rateLimiter.IsBlocked(clientIP)

		data := struct {
			Error                string
			IsBlocked            bool
			UsingDefaultPassword bool
		}{
			Error:                "",
			IsBlocked:            isBlocked,
			UsingDefaultPassword: !p.config.Web.DefaultPasswordChanged,
		}

		renderTemplate(w, "login.html", data)
		return
	}

	if r.Method == "POST" {
		// Check if IP is blocked
		if rateLimiter.IsBlocked(clientIP) {
			data := struct {
				Error                string
				IsBlocked            bool
				UsingDefaultPassword bool
			}{
				Error:                "Too many failed login attempts. Please try again later.",
				IsBlocked:            true,
				UsingDefaultPassword: !p.config.Web.DefaultPasswordChanged,
			}

			renderTemplate(w, "login.html", data)
			return
		}

		r.ParseForm()
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		// Validate credentials
		if username != "admin" || !ValidatePassword(password, p.config.Web.Password) {
			// Record failed attempt
			rateLimiter.RecordFailedAttempt(clientIP)

			data := struct {
				Error                string
				IsBlocked            bool
				UsingDefaultPassword bool
			}{
				Error:                "Invalid username or password",
				IsBlocked:            false,
				UsingDefaultPassword: !p.config.Web.DefaultPasswordChanged,
			}

			renderTemplate(w, "login.html", data)
			return
		}

		// Record successful login (clears failed attempts)
		rateLimiter.RecordSuccessfulLogin(clientIP)

		// Check if using default password (admin/admin)
		if !p.config.Web.DefaultPasswordChanged && ValidatePassword("admin", p.config.Web.Password) {
			// Force password change
			http.Redirect(w, r, "/change-password?force=true", http.StatusSeeOther)
			return
		}

		// Create session
		session, err := sessionManager.CreateSession("admin", p.config.Web.SessionTimeout)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    session.ID,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			MaxAge:   p.config.Web.SessionTimeout * 60,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (p *DDNSPilot) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	forced := r.URL.Query().Get("force") == "true"

	switch r.Method {
	case "GET":
		data := struct {
			Forced bool
		}{
			Forced: forced,
		}
		renderTemplate(w, "change-password.html", data)
	case "POST":
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")
		acknowledged := r.FormValue("security_acknowledged") == "on"

		if newPassword == "" {
			http.Error(w, "Password cannot be empty", http.StatusBadRequest)
			return
		}

		if newPassword != confirmPassword {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		if len(newPassword) < 8 {
			http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
			return
		}

		if newPassword == "admin" {
			http.Error(w, "Cannot use 'admin' as password", http.StatusBadRequest)
			return
		}

		if !acknowledged {
			http.Error(w, "You must acknowledge the security warnings", http.StatusBadRequest)
			return
		}

		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// Update configuration
		p.config.Web.Password = string(hashedPassword)
		p.config.Web.DefaultPasswordChanged = true
		p.config.Web.SecurityAcknowledged = acknowledged

		// Save configuration
		if err := p.config.save(); err != nil {
			log.Printf("Error saving config: %v", err)
			http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
			return
		}

		// If this was a forced change, create session and redirect to dashboard
		if forced {
			session, err := sessionManager.CreateSession("admin", p.config.Web.SessionTimeout)
			if err != nil {
				http.Error(w, "Failed to create session", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    session.ID,
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				MaxAge:   p.config.Web.SessionTimeout * 60,
			})
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *DDNSPilot) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	if cookie, err := r.Cookie("session_id"); err == nil {
		// Delete session
		sessionManager.DeleteSession(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (p *DDNSPilot) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Get current public IP for display
	currentIP, _ := p.ddns.GetPublicIP()

	// Check for update result messages
	updateResult := r.URL.Query().Get("update_result")
	var updateMessage, updateType string

	switch updateResult {
	case "success":
		updated := r.URL.Query().Get("updated")
		updateMessage = fmt.Sprintf("✅ Successfully updated %s record(s)", updated)
		updateType = "success"
	case "mixed":
		success := r.URL.Query().Get("success")
		errors := r.URL.Query().Get("errors")
		updateMessage = fmt.Sprintf("⚠️ Mixed results: %s successful, %s failed", success, errors)
		updateType = "warning"
	case "single_success":
		record := r.URL.Query().Get("record")
		ip := r.URL.Query().Get("ip")
		updateMessage = fmt.Sprintf("✅ Successfully updated %s to %s", record, ip)
		updateType = "success"
	case "single_error":
		record := r.URL.Query().Get("record")
		errorMsg := r.URL.Query().Get("error")
		updateMessage = fmt.Sprintf("❌ Failed to update %s: %s", record, errorMsg)
		updateType = "error"
	}

	data := struct {
		Records       []DDNSRecord
		CurrentIP     string
		Config        *AppConfig
		UpdateMessage string
		UpdateType    string
	}{
		Records:       p.config.Records,
		CurrentIP:     currentIP,
		Config:        p.config,
		UpdateMessage: updateMessage,
		UpdateType:    updateType,
	}

	renderTemplate(w, "index.html", data)
}

func (p *DDNSPilot) handleAddRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()

		record := DDNSRecord{
			RecordName: strings.TrimSpace(r.FormValue("record_name")),
			APIToken:   strings.TrimSpace(r.FormValue("api_token")),
			Proxied:    r.FormValue("proxied") == "true",
			Notes:      strings.TrimSpace(r.FormValue("notes")),
		}

		if record.RecordName == "" {
			http.Error(w, "Record name cannot be empty", http.StatusBadRequest)
			return
		}

		if record.APIToken == "" {
			http.Error(w, "API token cannot be empty", http.StatusBadRequest)
			return
		}

		// Check if record already exists
		if _, err := p.config.GetRecord(record.RecordName); err == nil {
			http.Error(w, "Record already exists", http.StatusConflict)
			return
		}

		// Try to auto-fill zone and record IDs
		zoneName, err := p.ddns.ExtractZoneName(record.RecordName)
		if err != nil {
			http.Error(w, "Invalid record name: "+err.Error(), http.StatusBadRequest)
			return
		}

		zoneID, err := p.ddns.GetZoneID(record.APIToken, zoneName)
		if err != nil {
			http.Error(w, "Failed to get zone ID: "+err.Error(), http.StatusBadRequest)
			return
		}
		record.ZoneID = zoneID

		recordID, err := p.ddns.GetRecordID(record.APIToken, record.ZoneID, record.RecordName)
		if err != nil {
			http.Error(w, "Failed to get record ID: "+err.Error(), http.StatusBadRequest)
			return
		}
		record.RecordID = recordID

		// Add the record
		p.config.AddRecord(record)

		if err := p.config.save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := struct {
		DefaultAPIToken string
	}{
		DefaultAPIToken: p.config.DefaultAPIToken,
	}

	renderTemplate(w, "add-record.html", data)
}

func (p *DDNSPilot) handleEditRecord(w http.ResponseWriter, r *http.Request) {
	recordName := r.URL.Query().Get("name")
	if recordName == "" {
		http.Error(w, "Record name required", http.StatusBadRequest)
		return
	}

	record, err := p.config.GetRecord(recordName)
	if err != nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()

		updatedRecord := *record
		updatedRecord.Proxied = r.FormValue("proxied") == "true"
		updatedRecord.Notes = strings.TrimSpace(r.FormValue("notes"))

		if err := p.config.UpdateRecord(recordName, updatedRecord); err != nil {
			http.Error(w, "Failed to update record: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := p.config.save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := struct {
		Record DDNSRecord
	}{
		Record: *record,
	}

	renderTemplate(w, "edit-record.html", data)
}

func (p *DDNSPilot) handleRemoveRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	recordName := r.FormValue("record_name")

	if err := p.config.RemoveRecord(recordName); err != nil {
		http.Error(w, "Failed to remove record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := p.config.save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *DDNSPilot) handleToggleRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	recordName := r.FormValue("record_name")

	record, err := p.config.GetRecord(recordName)
	if err != nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	// Toggle enabled status
	record.Enabled = !record.Enabled

	if err := p.config.UpdateRecord(recordName, *record); err != nil {
		http.Error(w, "Failed to update record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := p.config.save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *DDNSPilot) handleUpdateRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	results := p.ddns.UpdateAllRecords()

	// Log all results
	log.Printf("Update all records completed: %d results", len(results))
	for _, result := range results {
		if result.Success {
			log.Printf("✅ %s: %s (IP: %s)", result.RecordName, result.Message, result.NewIP)
		} else {
			log.Printf("❌ %s: %s", result.RecordName, result.Message)
		}
	}

	// For AJAX requests, return JSON response
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" || r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"results": results,
		})
		return
	}

	// For regular requests, redirect with a success message
	// We'll add a query parameter to show results on the main page
	successCount := 0
	errorCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			errorCount++
		}
	}

	if errorCount > 0 {
		http.Redirect(w, r, fmt.Sprintf("/?update_result=mixed&success=%d&errors=%d", successCount, errorCount), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/?update_result=success&updated=%d", successCount), http.StatusSeeOther)
	}
}

func (p *DDNSPilot) handleUpdateSingle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	recordName := r.FormValue("record_name")

	result := p.ddns.AutoUpdateRecord(recordName)

	// Log the result
	if result.Success {
		log.Printf("✅ Single update %s: %s (IP: %s)", result.RecordName, result.Message, result.NewIP)
	} else {
		log.Printf("❌ Single update %s: %s", result.RecordName, result.Message)
	}

	// For AJAX requests, return JSON response
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" || r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// For regular requests, redirect with result information
	if result.Success {
		http.Redirect(w, r, fmt.Sprintf("/?update_result=single_success&record=%s&ip=%s", result.RecordName, result.NewIP), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/?update_result=single_error&record=%s&error=%s", result.RecordName, result.Message), http.StatusSeeOther)
	}
}

func (p *DDNSPilot) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()

		// Parse update interval
		if intervalStr := r.FormValue("update_interval"); intervalStr != "" {
			if interval, err := strconv.Atoi(intervalStr); err == nil && interval >= 1 && interval <= 1440 {
				p.config.UpdateInterval = interval
			}
		}

		// Parse auto-update setting
		p.config.AutoUpdate = r.FormValue("auto_update") == "true"

		// Parse web port
		if portStr := r.FormValue("web_port"); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil && port >= 1 && port <= 65535 {
				p.config.Web.Port = port
			}
		}

		// Parse default API token
		if defaultToken := strings.TrimSpace(r.FormValue("default_api_token")); defaultToken != "" {
			p.config.DefaultAPIToken = defaultToken
		} else {
			// Allow clearing the default token
			p.config.DefaultAPIToken = ""
		}

		if err := p.config.save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := struct {
		Config *AppConfig
	}{
		Config: p.config,
	}

	renderTemplate(w, "settings.html", data)
}

func (p *DDNSPilot) handleStatsAPI(w http.ResponseWriter, r *http.Request) {
	// Get current public IP
	currentIP, _ := p.ddns.GetPublicIP()

	stats := map[string]interface{}{
		"current_ip":    currentIP,
		"total_records": len(p.config.Records),
		"enabled_records": func() int {
			count := 0
			for _, record := range p.config.Records {
				if record.Enabled {
					count++
				}
			}
			return count
		}(),
		"auto_update":     p.config.AutoUpdate,
		"update_interval": p.config.UpdateInterval,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (p *DDNSPilot) handleAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Return current config as JSON (sanitized)
		sanitizedConfig := struct {
			Records        []DDNSRecord `json:"records"`
			UpdateInterval int          `json:"update_interval"`
			AutoUpdate     bool         `json:"auto_update"`
		}{
			Records:        p.config.Records,
			UpdateInterval: p.config.UpdateInterval,
			AutoUpdate:     p.config.AutoUpdate,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sanitizedConfig)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
