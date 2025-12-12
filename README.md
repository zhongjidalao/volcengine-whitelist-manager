# Volcengine SSH IP Updater

<div align="center">

**自动更新火山引擎安全组 SSH 访问规则的智能工具**

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](#) | [简体中文](#)

</div>

---

## 📖 项目简介

Volcengine SSH IP Updater 是一个基于 Go 语言开发的自动化工具,用于实时监控您的公网 IP 地址变化,并自动更新火山引擎(Volcengine)安全组的 SSH 入站规则,确保只有您当前的 IP 地址可以访问服务器。

### 🎯 使用场景

- **动态 IP 环境**: 家庭宽带、移动办公等场景下 IP 地址频繁变化
- **安全加固**: 限制 SSH 访问来源,防止暴力破解和未授权访问
- **远程办公**: 自动适应不同网络环境,无需手动修改安全组规则
- **多端口管理**: 同时管理多个服务端口的访问控制

### ✨ 核心特性

- 🔄 **自动监控**: 定时检查公网 IP 变化(默认 15 分钟,可自定义)
- 🔐 **安全组自动更新**: 实时同步 IP 变化到火山引擎安全组规则
- 🌐 **Web 管理界面**: 提供可视化配置面板和日志监控
- 🚀 **多端口支持**: 一次配置多个端口(如 22,8080,3389),逗号分隔
- 📊 **完整日志记录**: 所有操作可追溯,支持分页查看和清空
- ⚡ **高性能**: Go 语言编写,资源占用低,响应速度快
- 📦 **零依赖部署**: 单一二进制文件,无需额外安装运行时环境
- 🔁 **智能重试**: 多个 IP 查询源自动切换,确保高可用性
- 🛡️ **容错设计**: 配置不完整时自动跳过,避免误操作

---

## 🏗️ 项目架构

```text
volcengine-updater/
├── cmd/
│   └── server/
│       └── main.go           # 应用程序入口点
├── internal/
│   ├── config/
│   │   └── db.go             # 数据库初始化和配置管理
│   ├── models/
│   │   └── models.go         # 数据模型定义(Settings, UpdateLog)
│   ├── service/
│   │   └── updater.go        # 核心业务逻辑(IP检测、安全组更新)
│   └── web/
│       └── handler.go        # Web 路由和 HTTP 处理器
├── templates/                # HTML 模板文件
│   ├── index.html            # 主仪表盘
│   ├── settings.html         # 配置页面
│   └── logs.html             # 日志查看页面
├── instance/                 # 运行时数据目录(自动创建)
│   └── config.db             # SQLite 数据库
├── go.mod                    # Go 模块依赖
├── go.sum                    # 依赖校验文件
└── README.md                 # 本文件
```

### 技术栈

- **Web 框架**: [Gin](https://github.com/gin-gonic/gin) - 高性能 HTTP 框架
- **任务调度**: [Cron v3](https://github.com/robfig/cron) - 可靠的定时任务调度器
- **数据库**: [GORM](https://gorm.io/) + SQLite - 轻量级数据持久化
- **云服务 SDK**: [Volcengine Go SDK](https://github.com/volcengine/volcengine-go-sdk) - 官方 SDK

---

## 🚀 快速开始

### 系统要求

- **编译环境**: Go 1.20 或更高版本
- **运行环境**: Linux / macOS / Windows
- **网络要求**: 能够访问火山引擎 API 和公网 IP 查询服务

### 安装步骤

#### 方法一: 从源码编译

```bash
# 1. 克隆仓库
git clone <repository-url>
cd volcengine-updater

# 2. 安装依赖
go mod tidy

# 3. 编译二进制文件
go build -o volcengine-updater cmd/server/main.go

# 4. 运行程序
./volcengine-updater
```

#### 方法二: 直接运行(开发模式)

```bash
go run cmd/server/main.go
```

### 初始配置

1. **启动服务**
   程序启动后,访问 `http://localhost:5000`

2. **进入设置页面**
   点击导航栏的 "Settings" 按钮

3. **填写配置信息**

   | 配置项 | 说明 | 示例 |
   |--------|------|------|
   | Access Key | 火山引擎访问密钥 | AKLT... |
   | Secret Key | 火山引擎私钥 | *** |
   | Region | 安全组所在区域 | `cn-beijing`, `cn-shanghai` |
   | Security Group ID | 目标安全组 ID | `sg-xxxxxx` |
   | SSH Port | 需要管理的端口 | `22` 或 `22,8080,3389` |
   | Check Interval | 检查间隔 | `15` (分钟) |
   | IP Services | IP 查询服务列表 | 默认已配置多个备用源 |

4. **保存并测试**
   点击 "Save Settings" 后,可点击主页的 "Run Now" 按钮立即触发一次更新

---

## 📋 使用指南

### Web 界面功能

#### 主仪表盘 (`/`)
- 显示当前配置概览
- 查看最近 10 条操作日志
- 显示下次自动运行时间
- 提供 "立即运行" 按钮

#### 设置页面 (`/settings`)
- 配置火山引擎凭证
- 设置检查间隔和端口
- 管理 IP 查询服务列表

#### 日志页面 (`/logs`)
- 分页查看所有操作日志
- 支持清空历史记录
- 显示 INFO/WARNING/ERROR 级别日志

### API 接口

```bash
# 获取最近 50 条日志
GET /api/logs

# 获取当前状态
GET /api/status

# 立即触发更新
POST /run_now

# 清空日志
POST /logs/clear
```

---

## ⚙️ 高级配置

### 多端口配置

在 "SSH Port" 字段中使用逗号分隔多个端口:

```
22,8080,3389,5000
```

程序会为每个端口创建独立的安全组规则。

### 自定义 IP 查询服务

默认使用以下服务(按顺序尝试):
- https://myip.ipip.net
- https://ddns.oray.com/checkip
- https://ip.3322.net
- https://v4.yinghualuo.cn/bejson

可在设置页面的 "IP Services" 字段添加自定义服务,每行一个 URL。

### 检查间隔时间

- 最小值: 60 秒
- 推荐值: 900 秒(15 分钟)
- 单位支持: 秒(seconds) / 分钟(minutes) / 小时(hours)

---

## 🔧 开发指南

### 本地开发

```bash
# 安装依赖
go mod tidy

# 运行开发服务器(热重载需配合工具如 air)
go run cmd/server/main.go

# 运行测试(如果有)
go test ./...

# 代码格式化
go fmt ./...
```

### 构建优化

```bash
# 编译优化版本(减小体积)
go build -ldflags="-s -w" -o volcengine-updater cmd/server/main.go

# 跨平台编译
GOOS=linux GOARCH=amd64 go build -o volcengine-updater-linux cmd/server/main.go
GOOS=windows GOARCH=amd64 go build -o volcengine-updater.exe cmd/server/main.go
```

---

## 🐛 故障排除

### 常见问题

**Q: 提示 "配置不完整" 无法更新?**
A: 确保 Access Key, Secret Key, Region, Security Group ID 都已正确填写。

**Q: 无法获取公网 IP?**
A: 检查网络连接,或在设置中添加更多备用 IP 查询服务。

**Q: 安全组规则更新失败?**
A: 检查以下几点:
- Access Key 是否有安全组修改权限
- Security Group ID 是否正确
- Region 配置是否与安全组所在区域一致

**Q: 数据库文件在哪里?**
A: 自动创建在 `instance/config.db`,与可执行文件同级目录。

**Q: 如何修改监听端口?**
A: 编辑 `cmd/server/main.go` 第 48 行,修改 `:5000` 为其他端口。

---

## 📊 日志说明

### 日志级别

- **INFO**: 正常操作记录(IP 检查、规则更新成功)
- **WARNING**: 警告信息(某个 IP 服务不可用、配置跳过)
- **ERROR**: 错误信息(API 调用失败、授权失败)

### 日志示例

```
[INFO] 开始IP检查...
[INFO] 当前公网IP: 123.45.67.89 (来源: https://myip.ipip.net)
[INFO] 端口 22: 撤销旧规则 111.22.33.44/32
[INFO] 端口 22: 添加新规则 123.45.67.89/32
[INFO] ✓ 端口 22: 已更新允许 123.45.67.89/32
```

---

## 🔒 安全建议

1. **凭证管理**: 不要将 Access Key 和 Secret Key 提交到版本控制系统
2. **最小权限**: 为程序创建专用的 RAM 用户,仅授予安全组修改权限
3. **端口限制**: 仅开放必要的端口,避免使用 `0.0.0.0/0` 规则
4. **日志审计**: 定期检查日志,发现异常操作
5. **HTTPS 访问**: 生产环境建议配置反向代理(Nginx)启用 HTTPS

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request!

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

## 📮 联系方式

如有问题或建议,请通过以下方式联系:

- 提交 [Issue](../../issues)
- 发起 [Discussion](../../discussions)

---

<div align="center">

**⭐ 如果这个项目对你有帮助,请给个 Star!**

Made with ❤️ by Go

</div>