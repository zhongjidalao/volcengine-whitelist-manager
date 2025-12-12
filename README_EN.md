# Volcengine Whitelist Manager

<div align="center">

**Intelligent Tool for Automatically Updating Volcengine Security Group Whitelist Access Rules**

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](README_EN.md) | [简体中文](README.md)

</div>

---

## 📖 Introduction

Volcengine Whitelist Manager is an automation tool developed in Go that monitors your public IP address changes in real-time and automatically updates Volcengine security group whitelist access rules, ensuring only your current IP address can access your servers.

### 🎯 Use Cases

- **Dynamic IP Environments**: Home broadband, mobile offices where IP addresses change frequently
- **Security Hardening**: Restrict service access sources to prevent brute force attacks and unauthorized access
- **Remote Work**: Automatically adapt to different network environments without manual security group rule modifications
- **Multi-Port Management**: Manage whitelist access control for multiple service ports simultaneously

### ✨ Key Features

- 🔄 **Automatic Monitoring**: Periodic public IP change detection (default 15 minutes, customizable)
- 🔐 **Auto Security Group Updates**: Real-time synchronization of IP changes to Volcengine security group rules
- 🌐 **Web Management Interface**: Visual configuration panel and log monitoring
- 🚀 **Multi-Port Support**: Configure multiple ports at once (e.g., 22,8080,3389), comma-separated
- 📊 **Complete Log Recording**: All operations are traceable with pagination support and clear function
- ⚡ **High Performance**: Written in Go, low resource consumption, fast response
- 📦 **Zero Dependency Deployment**: Single binary file, no additional runtime environment required
- 🔁 **Intelligent Retry**: Automatic switching between multiple IP query sources for high availability
- 🛡️ **Fault Tolerance Design**: Automatically skips when configuration is incomplete, avoiding misoperations

---

## 🏗️ Project Architecture

```text
volcengine-whitelist-manager/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── db.go             # Database initialization and configuration management
│   ├── models/
│   │   └── models.go         # Data model definitions (Settings, UpdateLog)
│   ├── service/
│   │   └── updater.go        # Core business logic (IP detection, security group updates)
│   └── web/
│       └── handler.go        # Web routing and HTTP handlers
├── templates/                # HTML template files
│   ├── index.html            # Main dashboard
│   ├── settings.html         # Configuration page
│   └── logs.html             # Log viewing page
├── instance/                 # Runtime data directory (auto-created)
│   └── config.db             # SQLite database
├── go.mod                    # Go module dependencies
├── go.sum                    # Dependency checksum file
└── README.md                 # This file
```

### Technology Stack

- **Web Framework**: [Gin](https://github.com/gin-gonic/gin) - High-performance HTTP framework
- **Task Scheduling**: [Cron v3](https://github.com/robfig/cron) - Reliable scheduled task scheduler
- **Database**: [GORM](https://gorm.io/) + SQLite - Lightweight data persistence
- **Cloud Service SDK**: [Volcengine Go SDK](https://github.com/volcengine/volcengine-go-sdk) - Official SDK

---

## 🚀 Quick Start

### System Requirements

- **Build Environment**: Go 1.20 or higher
- **Runtime Environment**: Linux / macOS / Windows
- **Network Requirements**: Access to Volcengine API and public IP query services

### Installation

#### Method 1: Build from Source

```bash
# 1. Clone repository
git clone <repository-url>
cd volcengine-whitelist-manager

# 2. Install dependencies
go mod tidy

# 3. Build binary
go build -o volcengine-whitelist-manager cmd/server/main.go

# 4. Run program
./volcengine-whitelist-manager
```

#### Method 2: Run Directly (Development Mode)

```bash
go run cmd/server/main.go
```

### Initial Configuration

1. **Start Service**
   After the program starts, visit `http://localhost:9877`

2. **Navigate to Settings Page**
   Click the "Settings" button in the navigation bar

3. **Fill in Configuration**

   | Configuration | Description | Example |
   |--------------|-------------|---------|
   | Access Key | Volcengine access key | AKLT... |
   | Secret Key | Volcengine secret key | *** |
   | Region | Security group region | `cn-beijing`, `cn-shanghai` |
   | Security Group ID | Target security group ID | `sg-xxxxxx` |
   | Ports | Ports to manage | `22` or `22,8080,3389` |
   | Check Interval | Check interval | `15` (minutes) |
   | IP Services | IP query service list | Multiple backup sources pre-configured |

4. **Save and Test**
   After clicking "Save Settings", you can click the "Run Now" button on the homepage to trigger an immediate update

---

## 📋 User Guide

### Web Interface Features

#### Main Dashboard (`/`)
- Display current configuration overview
- View recent 10 operation logs
- Show next automatic run time
- Provide "Run Now" button

#### Settings Page (`/settings`)
- Configure Volcengine credentials
- Set check interval and ports
- Manage IP query service list

#### Logs Page (`/logs`)
- Paginated view of all operation logs
- Support for clearing history
- Display INFO/WARNING/ERROR level logs

### API Endpoints

```bash
# Get recent 50 logs
GET /api/logs

# Get current status
GET /api/status

# Trigger immediate update
POST /run_now

# Clear logs
POST /logs/clear
```

---

## ⚙️ Advanced Configuration

### Multi-Port Configuration

Use comma-separated values in the "Ports" field:

```
22,8080,3389,5000
```

The program will create independent security group rules for each port.

### Custom IP Query Services

Default services used (attempted in order):
- https://myip.ipip.net
- https://ddns.oray.com/checkip
- https://ip.3322.net
- https://v4.yinghualuo.cn/bejson

You can add custom services in the "IP Services" field on the settings page, one URL per line.

### Check Interval Time

- Minimum: 60 seconds
- Recommended: 900 seconds (15 minutes)
- Unit support: seconds / minutes / hours

---

## 🔧 Development Guide

### Local Development

```bash
# Install dependencies
go mod tidy

# Run development server (hot reload requires tools like air)
go run cmd/server/main.go

# Run tests (if available)
go test ./...

# Code formatting
go fmt ./...
```

### Build Optimization

```bash
# Build optimized version (reduce size)
go build -ldflags="-s -w" -o volcengine-whitelist-manager cmd/server/main.go

# Cross-platform compilation
GOOS=linux GOARCH=amd64 go build -o volcengine-whitelist-manager-linux cmd/server/main.go
GOOS=windows GOARCH=amd64 go build -o volcengine-whitelist-manager.exe cmd/server/main.go
```

---

## 🐛 Troubleshooting

### Common Issues

**Q: Getting "Incomplete configuration" error?**
A: Ensure Access Key, Secret Key, Region, and Security Group ID are all correctly filled in.

**Q: Cannot get public IP?**
A: Check network connection or add more backup IP query services in settings.

**Q: Security group rule update failed?**
A: Check the following:
- Does the Access Key have security group modification permissions?
- Is the Security Group ID correct?
- Does the Region configuration match the security group's region?

**Q: Where is the database file?**
A: Automatically created at `instance/config.db`, in the same directory as the executable.

**Q: How to change the listening port?**
A: Edit line 47 in `cmd/server/main.go`, change `:9877` to another port.

---

## 📊 Log Description

### Log Levels

- **INFO**: Normal operation records (IP checks, successful rule updates)
- **WARNING**: Warning messages (IP service unavailable, configuration skipped)
- **ERROR**: Error messages (API call failures, authorization failures)

### Log Examples

```
[INFO] Starting IP check...
[INFO] Current public IP: 123.45.67.89 (source: https://myip.ipip.net)
[INFO] Port 22: Revoking old rule 111.22.33.44/32
[INFO] Port 22: Adding new rule 123.45.67.89/32
[INFO] ✓ Port 22: Updated to allow 123.45.67.89/32
```

---

## 🔒 Security Recommendations

1. **Credential Management**: Do not commit Access Key and Secret Key to version control systems
2. **Least Privilege**: Create a dedicated RAM user for the program with only security group modification permissions
3. **Port Restrictions**: Only open necessary ports, avoid using `0.0.0.0/0` rules
4. **Log Auditing**: Regularly check logs for abnormal operations
5. **HTTPS Access**: In production environments, consider configuring a reverse proxy (Nginx) with HTTPS enabled

---

## 🤝 Contributing

Issues and Pull Requests are welcome!

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

---

## 📮 Contact

For questions or suggestions, please contact us via:

- Submit an [Issue](../../issues)
- Start a [Discussion](../../discussions)

---

<div align="center">

**⭐ If this project helps you, please give it a Star!**

Made with ❤️ by Go

</div>
