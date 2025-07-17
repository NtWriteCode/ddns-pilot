package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// CloudFlareAPI represents API response structures
type CloudFlareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CloudFlareRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

type CloudFlareResponse struct {
	Success bool              `json:"success"`
	Errors  []CloudFlareError `json:"errors"`
	Result  json.RawMessage   `json:"result"`
}

type CloudFlareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// UpdateResult represents the result of a DNS update
type UpdateResult struct {
	RecordName string
	Success    bool
	OldIP      string
	NewIP      string
	Message    string
	UpdatedAt  time.Time
}

// DDNSManager handles DDNS operations
type DDNSManager struct {
	config *AppConfig
}

func NewDDNSManager(config *AppConfig) *DDNSManager {
	return &DDNSManager{config: config}
}

// GetPublicIP retrieves the current public IP address
func (dm *DDNSManager) GetPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("failed to get public IP: %v", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	ip := strings.TrimSpace(buf.String())
	if ip == "" {
		return "", fmt.Errorf("empty IP response")
	}

	return ip, nil
}

// GetDNSIP retrieves the current DNS IP for a record using dig
func (dm *DDNSManager) GetDNSIP(recordName string) (string, error) {
	cmd := exec.Command("dig", "+short", recordName, "@1.1.1.1")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query DNS: %v", err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", fmt.Errorf("no DNS record found")
	}

	return ip, nil
}

// ExtractZoneName extracts the zone name from a record name
func (dm *DDNSManager) ExtractZoneName(recordName string) (string, error) {
	parts := strings.Split(recordName, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid record name: %s", recordName)
	}
	return strings.Join(parts[len(parts)-2:], "."), nil
}

// GetZoneID retrieves the zone ID for a domain
func (dm *DDNSManager) GetZoneID(apiToken, zoneName string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zoneName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	var cfResp CloudFlareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if !cfResp.Success {
		return "", fmt.Errorf("CloudFlare API error: %v", cfResp.Errors)
	}

	var zones []CloudFlareZone
	if err := json.Unmarshal(cfResp.Result, &zones); err != nil {
		return "", fmt.Errorf("failed to parse zones: %v", err)
	}

	if len(zones) == 0 {
		return "", fmt.Errorf("zone not found: %s", zoneName)
	}

	return zones[0].ID, nil
}

// GetRecordID retrieves the record ID for a DNS record
func (dm *DDNSManager) GetRecordID(apiToken, zoneID, recordName string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s", zoneID, recordName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	var cfResp CloudFlareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if !cfResp.Success {
		return "", fmt.Errorf("CloudFlare API error: %v", cfResp.Errors)
	}

	var records []CloudFlareRecord
	if err := json.Unmarshal(cfResp.Result, &records); err != nil {
		return "", fmt.Errorf("failed to parse records: %v", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("record not found: %s", recordName)
	}

	return records[0].ID, nil
}

// UpdateRecord updates a single DNS record
func (dm *DDNSManager) UpdateRecord(record *DDNSRecord) *UpdateResult {
	result := &UpdateResult{
		RecordName: record.RecordName,
		UpdatedAt:  time.Now(),
	}

	log.Printf("🔄 Starting update for record: %s", record.RecordName)

	// Get current public IP
	newIP, err := dm.GetPublicIP()
	if err != nil {
		log.Printf("❌ Failed to get public IP for %s: %v", record.RecordName, err)
		result.Message = fmt.Sprintf("Failed to get public IP: %v", err)
		return result
	}
	result.NewIP = newIP
	log.Printf("📍 Current public IP: %s", newIP)

	// Get current DNS IP
	oldIP, err := dm.GetDNSIP(record.RecordName)
	if err != nil {
		// DNS query failed, but we can still try to update
		log.Printf("⚠️ Failed to query current DNS IP for %s: %v", record.RecordName, err)
		oldIP = "unknown"
	} else {
		log.Printf("🌐 Current DNS IP for %s: %s", record.RecordName, oldIP)
	}
	result.OldIP = oldIP

	// Check if update is needed
	if newIP == oldIP {
		log.Printf("✅ No update needed for %s - IP unchanged (%s)", record.RecordName, newIP)
		result.Success = true
		result.Message = "No update needed - IP unchanged"
		return result
	}

	log.Printf("🔄 IP change detected for %s: %s → %s", record.RecordName, oldIP, newIP)

	// Validate record configuration
	if record.ZoneID == "" {
		log.Printf("❌ Missing zone ID for %s", record.RecordName)
		result.Message = "Missing zone ID - record configuration incomplete"
		return result
	}
	if record.RecordID == "" {
		log.Printf("❌ Missing record ID for %s", record.RecordName)
		result.Message = "Missing record ID - record configuration incomplete"
		return result
	}
	if record.APIToken == "" {
		log.Printf("❌ Missing API token for %s", record.RecordName)
		result.Message = "Missing API token - record configuration incomplete"
		return result
	}

	// Update the DNS record via CloudFlare API
	updateData := map[string]interface{}{
		"type":    "A",
		"name":    record.RecordName,
		"content": newIP,
		"ttl":     300,
		"proxied": record.Proxied,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		log.Printf("❌ Failed to marshal update data for %s: %v", record.RecordName, err)
		result.Message = fmt.Sprintf("Failed to marshal update data: %v", err)
		return result
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", record.ZoneID, record.RecordID)
	log.Printf("🌐 Making API request to update %s: %s", record.RecordName, url)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Failed to create HTTP request for %s: %v", record.RecordName, err)
		result.Message = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	req.Header.Set("Authorization", "Bearer "+record.APIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ API request failed for %s: %v", record.RecordName, err)
		result.Message = fmt.Sprintf("API request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	log.Printf("📡 API response status for %s: %d", record.RecordName, resp.StatusCode)

	var cfResp CloudFlareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		log.Printf("❌ Failed to decode API response for %s: %v", record.RecordName, err)
		result.Message = fmt.Sprintf("Failed to decode response: %v", err)
		return result
	}

	if !cfResp.Success {
		log.Printf("❌ CloudFlare API returned error for %s: %v", record.RecordName, cfResp.Errors)
		result.Message = fmt.Sprintf("CloudFlare API error: %v", cfResp.Errors)
		return result
	}

	// Update succeeded
	log.Printf("✅ Successfully updated %s: %s → %s", record.RecordName, oldIP, newIP)
	result.Success = true
	result.Message = "DNS record updated successfully"

	// Update the record's last IP and timestamp
	record.LastIP = newIP
	record.LastUpdated = result.UpdatedAt.Format(time.RFC3339)

	return result
}

// UpdateAllRecords updates all enabled DNS records
func (dm *DDNSManager) UpdateAllRecords() []*UpdateResult {
	var results []*UpdateResult

	for i := range dm.config.Records {
		record := &dm.config.Records[i]
		if !record.Enabled {
			continue
		}

		result := dm.UpdateRecord(record)
		results = append(results, result)
	}

	// Save config to persist last IP and update times
	if err := dm.config.save(); err != nil {
		// Add error result for config save failure
		results = append(results, &UpdateResult{
			RecordName: "config",
			Success:    false,
			Message:    fmt.Sprintf("Failed to save config: %v", err),
			UpdatedAt:  time.Now(),
		})
	}

	return results
}

// AutoUpdateRecord automatically updates a specific record (used for scheduled updates)
func (dm *DDNSManager) AutoUpdateRecord(recordName string) *UpdateResult {
	record, err := dm.config.GetRecord(recordName)
	if err != nil {
		return &UpdateResult{
			RecordName: recordName,
			Success:    false,
			Message:    fmt.Sprintf("Record not found: %v", err),
			UpdatedAt:  time.Now(),
		}
	}

	if !record.Enabled {
		return &UpdateResult{
			RecordName: recordName,
			Success:    true,
			Message:    "Record disabled - skipped",
			UpdatedAt:  time.Now(),
		}
	}

	result := dm.UpdateRecord(record)

	// Save config to persist updates
	if result.Success {
		if err := dm.config.save(); err != nil {
			result.Message += fmt.Sprintf(" (Warning: failed to save config: %v)", err)
		}
	}

	return result
}

// ValidateRecord validates a DNS record configuration by testing API access
func (dm *DDNSManager) ValidateRecord(record DDNSRecord) error {
	// Test API token by getting zone info
	zoneName, err := dm.ExtractZoneName(record.RecordName)
	if err != nil {
		return fmt.Errorf("invalid record name: %v", err)
	}

	// Test zone access
	if record.ZoneID == "" {
		zoneID, err := dm.GetZoneID(record.APIToken, zoneName)
		if err != nil {
			return fmt.Errorf("failed to get zone ID: %v", err)
		}
		record.ZoneID = zoneID
	}

	// Test record access
	if record.RecordID == "" {
		recordID, err := dm.GetRecordID(record.APIToken, record.ZoneID, record.RecordName)
		if err != nil {
			return fmt.Errorf("failed to get record ID: %v", err)
		}
		record.RecordID = recordID
	}

	return nil
}
