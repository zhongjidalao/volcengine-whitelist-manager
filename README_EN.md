# Whitelist Manager

<div align="center">

**Intelligent Tool for Automatically Updating Volcengine / AWS Lightsail / AWS EC2 / Tencent Cloud CVM Whitelist Access Rules**

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](README_EN.md) | [简体中文](README.md)

</div>

---

## 📖 Introduction

Whitelist Manager is an automation tool developed in Go that monitors public IP changes in real-time and automatically updates cloud firewall whitelist rules, ensuring only your current IP can access your servers.

### 🎯 Use Cases

- **Dynamic IP Environments**: Home broadband, mobile offices where IP addresses change frequently
- **Security Hardening**: Restrict service access sources to prevent brute force attacks and unauthorized access
- **Remote Work**: Automatically adapt to different network environments without manual security group rule modifications
- **Multi-Port Management**: Manage whitelist access control for multiple service ports simultaneously
- **Multi-Cloud Support**: Manage Volcengine security groups, AWS Lightsail, AWS EC2, and Tencent Cloud CVM rules in one workflow

### ✨ Key Features

- 🔄 **Automatic Monitoring**: Periodic public IP change detection (default 15 minutes, customizable)
- 🔐 **Auto Whitelist Updates**: Real-time synchronization of IP changes to cloud firewall rules
- ☁️ **Multi-Provider Support**: Supports Volcengine, AWS Lightsail, AWS EC2, and Tencent Cloud CVM
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
whitelist-manager/
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
- **Cloud Service SDK**: [Volcengine Go SDK](https://github.com/volcengine/volcengine-go-sdk), [AWS SDK for Go](https://github.com/aws/aws-sdk-go), [Tencent Cloud SDK for Go](https://github.com/TencentCloud/tencentcloud-sdk-go)

---

## 🚀 Quick Start

### System Requirements

- **Build Environment**: Go 1.20 or higher (required only for source build or development mode)
- **Runtime Environment**: Linux / macOS
- **Network Requirements**: Access to cloud provider APIs and public IP query services

### Installation

#### Method 1: Build from Source

```bash
# 1. Clone repository
git clone <repository-url>
cd whitelist-manager

# 2. Install dependencies
go mod tidy

# 3. Build binary
go build -o whitelist-manager cmd/server/main.go

# 4. Run program
./whitelist-manager
```

#### Method 2: Run Directly (Development Mode)

```bash
go run cmd/server/main.go
```

#### Method 3: Download Executable from Release and Run

```bash
# 1. Open the latest release page and download the executable for Linux/macOS
# <repository-url>/releases/latest

# 2. Linux/macOS: make it executable and start
chmod +x whitelist-manager
./whitelist-manager
```

### Initial Configuration

1. **Start Service**
   After the program starts, visit `http://localhost:9877`

2. **Navigate to Settings Page**
   Click the "Settings" button in the navigation bar

3. **Fill in Configuration**

   | Configuration | Description | Example |
   |--------------|-------------|---------|
   | Providers | Cloud providers (multi-select) | `volcengine` + `aws` + `aws-ec2` + `tencent` |
   | Volcengine Access Key | Volcengine API access key | `AKLT...` |
   | Volcengine Secret Key | Volcengine API secret key | *** |
   | Volcengine Region | Volcengine region | `cn-beijing` |
   | Volcengine Security Group ID | Volcengine security group ID | `sg-xxxxxx` |
   | Volcengine Ports | Volcengine managed ports (comma-separated) | `22,3389` |
   | AWS Access Key | AWS API access key | `AKIA...` |
   | AWS Secret Key | AWS API secret key | *** |
   | AWS Region | Lightsail region | `ap-northeast-1` |
   | AWS Instance Name | Lightsail instance name | `my-lightsail-instance` |
   | AWS Ports | AWS managed ports (comma-separated) | `22,80,443` |
   | AWS EC2 Security Group ID | EC2 security group ID | `sg-abcdef123456` |
   | AWS EC2 Ports | AWS EC2 managed ports (comma-separated) | `22,443` |
   | Tencent SecretId | Tencent Cloud API SecretId | `AKID...` |
   | Tencent SecretKey | Tencent Cloud API SecretKey | *** |
   | Tencent Region | Tencent Cloud region | `ap-guangzhou` |
   | Tencent Security Group ID | Tencent Cloud security group ID | `sg-xxxxxx` |
   | Tencent Ports | Tencent managed ports (comma-separated) | `22,3389` |
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
- Multi-select one or more cloud providers
- Configure Volcengine, AWS, and Tencent Cloud credentials/resources separately
- Configure provider-specific port lists and check interval
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

### Multi-Provider Port Configuration

You can enable multiple providers at the same time and define separate ports:

```
Volcengine Ports: 22,3389
AWS Lightsail Ports: 22,80,443
AWS EC2 Ports: 22,443
Tencent Cloud Ports: 22,3389
```

The program applies whitelist rules independently for each provider.

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

### Log Retention Policy

- Automatically removes logs older than 15 days before each scheduled/manual run
- You can still clear all logs manually from the logs page

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
go build -ldflags="-s -w" -o whitelist-manager cmd/server/main.go

# Cross-platform compilation
GOOS=linux GOARCH=amd64 go build -o whitelist-manager-linux cmd/server/main.go
GOOS=linux GOARCH=arm64 go build -o whitelist-manager-linux-arm64 cmd/server/main.go
```

---

## 🐛 Troubleshooting

### Common Issues

**Q: Getting "Incomplete configuration" error?**
A: Make sure at least one provider is selected, then verify required fields for each selected provider:
- Volcengine: `Access Key` / `Secret Key` / `Region` / `Security Group ID` / `Volcengine Ports`
- AWS: `AWS Access Key` / `AWS Secret Key` / `AWS Region` / `AWS Instance Name` / `AWS Ports`
- Tencent Cloud: `SecretId` / `SecretKey` / `Region` / `Security Group ID` / `Tencent Ports`

**Q: Cannot get public IP?**
A: Check network connection or add more backup IP query services in settings.

**Q: Security group rule update failed?**
A: Check the following:
- Does the Access Key have security group modification permissions?
- Is the Security Group ID correct?
- Does the Region configuration match the security group's region?

**Q: AWS returns `DoesNotExist` / `The Instance does not exist`?**
A: The instance cannot be found under the current AK/SK + Region. Check:
- Is `AWS Region` exactly where the instance is deployed?
- Is `AWS Instance Name` exactly correct (case-sensitive)?
- Does the AK/SK belong to the correct AWS account?

**Q: AWS EC2 rule update failed?**
A: Check:
- Is `AWS EC2 Security Group ID` correct?
- Does the AK/SK have EC2 security group read/write permissions?
- Does `AWS Region` match the security group's actual region?

**Q: Volcengine returns `SignatureDoesNotMatch`?**
A: Signature verification failed. Check:
- Volcengine AK/SK are correct (no leading/trailing spaces)
- The AK/SK are still active and valid
- Server system time is synchronized (NTP)

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
[INFO] Starting IP check (providers=volcengine,aws)...
[INFO] Auto log cleanup completed: deleted 8 logs older than 15 days
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
