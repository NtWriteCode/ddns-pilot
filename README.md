# 🚁 DDNS Pilot

> **CloudFlare Dynamic DNS Manager with Web Interface**

A single **native binary** that provides both CLI and web interface for managing CloudFlare Dynamic DNS records. No Docker, no containers, no complexity - just drop the binary and run.

## ✨ Why This Exists

Most DDNS solutions are either too complex, require specific dependencies, or lack a modern interface. DDNS Pilot gives you both CLI convenience and web management in one lightweight binary.

**Need just CLI updates?** → Use the built-in CLI commands  
**Want a web interface?** → Start the web server mode  
**Want both?** → You're in the right place

## 🚀 Quick Start

```bash
# Download and run
wget https://github.com/NtWriteCode/ddns-pilot/releases/latest/download/ddns-pilot
chmod +x ddns-pilot

# CLI Mode - Add your first record
./ddns-pilot --add

# Web Mode - Start web interface (default)
./ddns-pilot
# Open browser → http://localhost:8082
# Login: admin / admin (changeable in settings)
```

That's it. No configuration files required to get started.

## ⚡ Features

### 🖥️ Web Interface
- **Modern Dashboard** - Clean, responsive web interface
- **Record Management** - Add, edit, enable/disable DNS records  
- **Real-time Updates** - Manual and automatic DNS updates
- **Settings Panel** - Configure auto-update intervals and preferences
- **Secure by Design** - Session-based auth, bcrypt passwords, rate limiting

### 📱 CLI Interface  
- **Interactive Setup** - Add records with guided prompts
- **Batch Updates** - Update all records with a single command
- **List & Status** - View all configured records and their status
- **Cron-friendly** - Perfect for scheduled updates

### 🔧 Core Features
- **CloudFlare API** - Full integration with CloudFlare DNS API
- **Multiple Records** - Manage unlimited DNS records
- **Auto-Detection** - Automatic zone and record ID lookup
- **Proxy Support** - Enable/disable CloudFlare proxy (orange cloud)
- **Update Tracking** - Track last IP and update timestamps

## 📋 Requirements

- **CloudFlare Account** with API token
- **DNS Records** already created in CloudFlare
- **Network Access** to CloudFlare API and IP detection services

No additional dependencies, databases, or services required.

## 🏗️ Installation

### Download Binary
```bash
# Intel/AMD (x64)
wget https://github.com/NtWriteCode/ddns-pilot/releases/latest/download/ddns-pilot-amd64 -O ddns-pilot

# ARM64 (Pi 4, Apple Silicon servers)
wget https://github.com/NtWriteCode/ddns-pilot/releases/latest/download/ddns-pilot-arm64 -O ddns-pilot

# ARM (Pi Zero, older Pi)
wget https://github.com/NtWriteCode/ddns-pilot/releases/latest/download/ddns-pilot-arm -O ddns-pilot

# macOS Intel
wget https://github.com/NtWriteCode/ddns-pilot/releases/latest/download/ddns-pilot-darwin-amd64 -O ddns-pilot

# macOS Apple Silicon
wget https://github.com/NtWriteCode/ddns-pilot/releases/latest/download/ddns-pilot-darwin-arm64 -O ddns-pilot

chmod +x ddns-pilot
```

### Build from Source
```bash
git clone https://github.com/NtWriteCode/ddns-pilot.git
cd ddns-pilot
go build -o ddns-pilot .
```

## 🎯 Usage

### CLI Mode

```bash
# Add a new DNS record
./ddns-pilot --add

# Update all enabled records
./ddns-pilot --update

# List all configured records  
./ddns-pilot --list

# Show help
./ddns-pilot --help
```

### Web Mode

```bash
# Start web interface (default mode)
./ddns-pilot

# Use custom port
PORT=8080 ./ddns-pilot
```

### Setup Process

1. **Get CloudFlare API Token**
   - Go to CloudFlare Dashboard → My Profile → API Tokens
   - Create token with `Zone:Read` and `DNS:Edit` permissions
   - Copy the token

2. **Create DNS Record**
   - In CloudFlare Dashboard, add an A record for your domain
   - Set any initial IP (it will be updated automatically)

3. **Add to DDNS Pilot**
   - CLI: `./ddns-pilot --add`
   - Web: Add Record button in interface
   - Enter record name (e.g., `home.example.com`) and API token

4. **Update DNS**
   - CLI: `./ddns-pilot --update`
   - Web: Update buttons in interface

## ⚙️ Configuration

- **Config file**: `ddns-pilot.json` (auto-created)
- **Web port**: `8082` (use `PORT=8081` to change)
- **Session timeout**: `60 minutes` (configurable in web interface)
- **Auto-update**: `Disabled by default` (configurable)

### Configuration Structure
```json
{
  "records": [
    {
      "record_name": "home.example.com",
      "api_token": "your_api_token",
      "proxied": false,
      "zone_id": "auto_detected",
      "record_id": "auto_detected",
      "enabled": true,
      "notes": "Home server"
    }
  ],
  "web": {
    "port": 8082,
    "password": "hashed_password"
  },
  "update_interval": 5,
  "auto_update": false
}
```

## 🔒 Security

**For local network use.** This tool manages your CloudFlare API tokens and should be treated securely.

- ✅ **Bcrypt password hashing**
- ✅ **Session-based authentication**  
- ✅ **Rate limiting** (5 attempts = 15min block)
- ✅ **Secure cookie flags**
- ✅ **Input validation** and **XSS protection**
- ✅ **API token encryption** in config files

### Security Best Practices
- **Change default password** immediately
- **Use on trusted networks only** (localhost/LAN)
- **Rotate API tokens** regularly
- **Run on-demand** for maximum security
- **Backup your configuration** securely

## 🛠️ Troubleshooting

**API Token Issues:**
```bash
# Test your token
curl -X GET "https://api.cloudflare.com/client/v4/zones" \
     -H "Authorization: Bearer YOUR_TOKEN"
```

**DNS Record Not Found:**
- Verify the record exists in CloudFlare Dashboard
- Check that record name exactly matches (including subdomain)
- Ensure API token has DNS:Edit permissions for the zone

**Network Issues:**
```bash
# Test IP detection
curl https://api.ipify.org

# Test DNS resolution
dig +short your-record.example.com @1.1.1.1
```

**Web Interface Not Accessible:**
```bash
# Check if port is available
netstat -tlnp | grep 8082

# Try different port
PORT=8083 ./ddns-pilot
```

## 🏗️ Architecture

```
ddns-pilot           (single binary)
├── CLI Mode         (command-line interface)
├── Web Mode         (HTTP server + HTML interface)
├── DDNS Engine      (CloudFlare API integration)
├── Config Manager   (JSON configuration)
└── Auto-Update      (background scheduler)
```

**Files:**
- `main.go` - Application entry point and CLI handling
- `config.go` - Configuration management and persistence
- `ddns.go` - CloudFlare API integration and DNS logic
- `handlers.go` - HTTP request handlers for web interface
- `templates.go` - HTML templates for web interface

## 📊 API

DDNS Pilot provides a simple JSON API when running in web mode:

- `GET /api` - Get current configuration (sanitized)
- `GET /api/stats` - Get statistics (IP, record counts, etc.)
- `POST /update-records` - Trigger update of all records
- `POST /update-single` - Update a specific record

## 🚀 Roadmap

- [ ] **IPv6 support**
- [ ] **Multiple IP sources** (custom URLs, interfaces)
- [ ] **Webhook notifications** (Discord, Slack, etc.)
- [ ] **Config import/export**
- [ ] **CloudFlare Analytics** integration
- [ ] **Docker image** (optional)

## 🤝 Contributing

This project prioritizes **simplicity and reliability**. PRs welcome for:
- Bug fixes
- Security improvements  
- Performance optimizations
- Documentation improvements

Please **avoid** adding:
- Heavy dependencies
- Complex frameworks
- Enterprise features that complicate the core use case

## 📝 License

MIT License - use it however you want.

---

**⭐ Star this repo if you find it useful!**

*Built with ❤️ for people who just want simple, reliable Dynamic DNS without the complexity.* 