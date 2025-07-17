package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// DDNSRecord represents a single DNS record configuration
type DDNSRecord struct {
	APIToken   string `json:"api_token"`
	RecordName string `json:"record_name"`
	Proxied    bool   `json:"proxied"`
	ZoneID     string `json:"zone_id"`
	RecordID   string `json:"record_id"`
	// Additional fields for enhanced functionality
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	LastIP      string `json:"last_ip"`
	Notes       string `json:"notes"`
}

// WebConfig represents web interface configuration
type WebConfig struct {
	Port                   int    `json:"port"`
	Password               string `json:"password"`
	SessionTimeout         int    `json:"session_timeout"`          // Minutes
	DefaultPasswordChanged bool   `json:"default_password_changed"` // Track if admin/admin was changed
	SecurityAcknowledged   bool   `json:"security_acknowledged"`    // Track if user acknowledged security warnings
}

// AppConfig represents the complete application configuration
type AppConfig struct {
	Records []DDNSRecord `json:"records"`
	Web     WebConfig    `json:"web"`

	// Update settings
	UpdateInterval int  `json:"update_interval"` // Minutes
	AutoUpdate     bool `json:"auto_update"`     // Enable automatic updates

	// Default CloudFlare API Token for new records
	DefaultAPIToken string `json:"default_api_token"`
}

// Session represents an active user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionManager handles user sessions
type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

// Simple rate limiting for login attempts
type LoginAttempt struct {
	FailedAttempts int
	LastAttempt    time.Time
	BlockedUntil   time.Time
}

type RateLimiter struct {
	attempts map[string]*LoginAttempt
	mutex    sync.RWMutex
}

// Global instances
var sessionManager = &SessionManager{
	sessions: make(map[string]*Session),
}

var rateLimiter = &RateLimiter{
	attempts: make(map[string]*LoginAttempt),
}

const defaultConfigPath = "ddns-pilot.json"

func loadConfig() (*AppConfig, error) {
	config := &AppConfig{
		Records: []DDNSRecord{},
		Web: WebConfig{
			Port:           8082,
			Password:       "admin",
			SessionTimeout: 60,
		},
		UpdateInterval: 5, // 5 minutes default
		AutoUpdate:     false,
	}

	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		// Hash the default password before returning
		if hashedPassword, err := HashPassword(config.Web.Password); err == nil {
			config.Web.Password = hashedPassword
		}
		return config, nil
	}

	data, err := os.ReadFile(defaultConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Migrate old configs and set defaults
	if config.Web.Port == 0 {
		config.Web.Port = 8082
	}
	if config.Web.SessionTimeout == 0 {
		config.Web.SessionTimeout = 60
	}
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 5
	}

	// SECURITY: Migrate plaintext passwords to hashed passwords
	if !strings.HasPrefix(config.Web.Password, "$2a$") && !strings.HasPrefix(config.Web.Password, "$2b$") {
		if hashedPassword, err := HashPassword(config.Web.Password); err == nil {
			config.Web.Password = hashedPassword
			if saveErr := config.save(); saveErr != nil {
				fmt.Printf("Warning: Failed to save hashed password to config: %v\n", saveErr)
			}
		} else {
			fmt.Printf("Warning: Failed to hash password: %v\n", err)
		}
	}

	return config, nil
}

func (c *AppConfig) save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(defaultConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func ensureConfigDir() error {
	dir := filepath.Dir(defaultConfigPath)
	return os.MkdirAll(dir, 0755)
}

// Rate limiter methods
func (rl *RateLimiter) IsBlocked(ip string) bool {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	attempt, exists := rl.attempts[ip]
	if !exists {
		return false
	}

	return time.Now().Before(attempt.BlockedUntil)
}

func (rl *RateLimiter) RecordFailedAttempt(ip string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	attempt, exists := rl.attempts[ip]
	if !exists {
		attempt = &LoginAttempt{}
		rl.attempts[ip] = attempt
	}

	attempt.FailedAttempts++
	attempt.LastAttempt = time.Now()

	// Block for 15 minutes after 5 failed attempts
	if attempt.FailedAttempts >= 5 {
		attempt.BlockedUntil = time.Now().Add(15 * time.Minute)
	}
}

func (rl *RateLimiter) RecordSuccessfulLogin(ip string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	delete(rl.attempts, ip)
}

func (rl *RateLimiter) CleanupOldAttempts() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	for ip, attempt := range rl.attempts {
		if attempt.LastAttempt.Before(cutoff) && time.Now().After(attempt.BlockedUntil) {
			delete(rl.attempts, ip)
		}
	}
}

// Session management functions
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (sm *SessionManager) CreateSession(userID string, timeoutMinutes int) (*Session, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(timeoutMinutes) * time.Minute),
	}

	sm.sessions[sessionID] = session
	return session, nil
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists || time.Now().After(session.ExpiresAt) {
		if exists {
			delete(sm.sessions, sessionID)
		}
		return nil, false
	}

	return session, true
}

func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.sessions, sessionID)
}

func (sm *SessionManager) CleanupExpiredSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, id)
		}
	}
}

// Password utilities
func ValidatePassword(provided, stored string) bool {
	if !strings.HasPrefix(stored, "$2a$") && !strings.HasPrefix(stored, "$2b$") {
		return provided == stored
	}

	err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(provided))
	return err == nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// DDNS record management
func (c *AppConfig) AddRecord(record DDNSRecord) {
	record.CreatedAt = time.Now().Format(time.RFC3339)
	record.Enabled = true
	c.Records = append(c.Records, record)
}

func (c *AppConfig) RemoveRecord(recordName string) error {
	for i, record := range c.Records {
		if record.RecordName == recordName {
			c.Records = append(c.Records[:i], c.Records[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("record not found: %s", recordName)
}

func (c *AppConfig) UpdateRecord(recordName string, updatedRecord DDNSRecord) error {
	for i, record := range c.Records {
		if record.RecordName == recordName {
			// Preserve creation time
			updatedRecord.CreatedAt = record.CreatedAt
			updatedRecord.LastUpdated = time.Now().Format(time.RFC3339)
			c.Records[i] = updatedRecord
			return nil
		}
	}
	return fmt.Errorf("record not found: %s", recordName)
}

func (c *AppConfig) GetRecord(recordName string) (*DDNSRecord, error) {
	for i, record := range c.Records {
		if record.RecordName == recordName {
			return &c.Records[i], nil
		}
	}
	return nil, fmt.Errorf("record not found: %s", recordName)
}
