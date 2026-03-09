# Whitelist Manager

<div align="center">

**自动更新 Volcengine / AWS Lightsail / AWS EC2 / Tencent Cloud CVM 白名单访问规则的智能工具**

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](README_EN.md) | [简体中文](README.md)

</div>

---

## 📖 项目简介

Whitelist Manager 是一个基于 Go 语言开发的自动化工具,用于实时监控公网 IP 地址变化,并自动更新云防火墙白名单规则,确保只有当前 IP 地址可以访问服务器。

### 🎯 使用场景

- **动态 IP 环境**: 家庭宽带、移动办公等场景下 IP 地址频繁变化
- **安全加固**: 限制服务访问来源,防止暴力破解和未授权访问
- **远程办公**: 自动适应不同网络环境,无需手动修改安全组规则
- **多端口管理**: 同时管理多个服务端口的白名单访问控制
- **多云支持**: 统一管理 Volcengine 安全组、AWS Lightsail、AWS EC2 与 Tencent Cloud CVM 规则

### ✨ 核心特性

- 🔄 **自动监控**: 定时检查公网 IP 变化(默认 15 分钟,可自定义)
- 🔐 **白名单自动更新**: 实时同步 IP 变化到云防火墙规则
- ☁️ **多供应商支持**: 支持 Volcengine、AWS Lightsail、AWS EC2、Tencent Cloud CVM
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
whitelist-manager/
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
- **云服务 SDK**: [Volcengine Go SDK](https://github.com/volcengine/volcengine-go-sdk), [AWS SDK for Go](https://github.com/aws/aws-sdk-go), [Tencent Cloud SDK for Go](https://github.com/TencentCloud/tencentcloud-sdk-go)

---

## 🚀 快速开始

### 系统要求

- **编译环境**: Go 1.20 或更高版本 (仅源码编译或开发模式需要)
- **运行环境**: Linux / macOS
- **网络要求**: 能够访问云厂商 API 和公网 IP 查询服务

### 安装步骤

#### 方法一: 从源码编译

```bash
# 1. 克隆仓库
git clone <repository-url>
cd whitelist-manager

# 2. 安装依赖
go mod tidy

# 3. 编译二进制文件
go build -o whitelist-manager cmd/server/main.go

# 4. 运行程序
./whitelist-manager
```

#### 方法二: 直接运行(开发模式)

```bash
go run cmd/server/main.go
```

#### 方法三: 从 Release 下载可执行文件运行

```bash
# 1. 打开最新发布页并下载 Linux/macOS 对应的可执行文件
# <repository-url>/releases/latest

# 2. Linux/macOS: 添加执行权限并启动
chmod +x whitelist-manager
./whitelist-manager
```

### 初始配置

1. **启动服务**
   程序启动后,访问 `http://localhost:9877`

2. **进入设置页面**
   点击导航栏的 "Settings" 按钮

3. **填写配置信息**

   | 配置项 | 说明 | 示例 |
   |--------|------|------|
   | Providers | 云供应商(可多选) | `volcengine` + `aws` + `aws-ec2` + `tencent` |
   | Volcengine Access Key | Volcengine 访问密钥 | `AKLT...` |
   | Volcengine Secret Key | Volcengine 私钥 | *** |
   | Volcengine Region | Volcengine 区域 | `cn-beijing` |
   | Volcengine Security Group ID | Volcengine 安全组 ID | `sg-xxxxxx` |
   | Volcengine Ports | Volcengine 管理端口(逗号分隔) | `22,3389` |
   | AWS Access Key | AWS 访问密钥 | `AKIA...` |
   | AWS Secret Key | AWS 私钥 | *** |
   | AWS Region | Lightsail 区域 | `ap-northeast-1` |
   | AWS Instance Name | Lightsail 实例名称 | `my-lightsail-instance` |
   | AWS Ports | AWS 管理端口(逗号分隔) | `22,80,443` |
   | AWS EC2 Security Group ID | EC2 安全组 ID | `sg-abcdef123456` |
   | AWS EC2 Ports | AWS EC2 管理端口(逗号分隔) | `22,443` |
   | Tencent SecretId | 腾讯云 API SecretId | `AKID...` |
   | Tencent SecretKey | 腾讯云 API SecretKey | *** |
   | Tencent Region | 腾讯云区域 | `ap-guangzhou` |
   | Tencent Security Group ID | 腾讯云安全组 ID | `sg-xxxxxx` |
   | Tencent Ports | 腾讯云管理端口(逗号分隔) | `22,3389` |
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
- 复选选择一个或多个云供应商
- 分别配置 Volcengine、AWS 与腾讯云的凭证和资源
- 分别配置各供应商端口与检查间隔
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

### 多供应商独立端口配置

可同时启用多个供应商，并分别设置端口:

```
Volcengine Ports: 22,3389
AWS Lightsail Ports: 22,80,443
AWS EC2 Ports: 22,443
Tencent Cloud Ports: 22,3389
```

程序会按供应商分别处理端口白名单，互不影响。

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

### 日志保留策略

- 每次任务执行前自动清理 15 天前日志
- 可在日志页面手动清空所有日志

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
go build -ldflags="-s -w" -o whitelist-manager cmd/server/main.go

# 跨平台编译
GOOS=linux GOARCH=amd64 go build -o whitelist-manager-linux cmd/server/main.go
GOOS=linux GOARCH=arm64 go build -o whitelist-manager-linux-arm64 cmd/server/main.go
```

---

## 🐛 故障排除

### 常见问题

**Q: 提示 "配置不完整" 无法更新?**
A: 先确认至少勾选了一个供应商，然后检查该供应商对应字段是否完整:
- Volcengine: `Access Key` / `Secret Key` / `Region` / `Security Group ID` / `Volcengine Ports`
- AWS: `AWS Access Key` / `AWS Secret Key` / `AWS Region` / `AWS Instance Name` / `AWS Ports`
- Tencent Cloud: `SecretId` / `SecretKey` / `Region` / `Security Group ID` / `Tencent Ports`

**Q: 无法获取公网 IP?**
A: 检查网络连接,或在设置中添加更多备用 IP 查询服务。

**Q: 安全组规则更新失败?**
A: 检查以下几点:
- Access Key 是否有安全组修改权限
- Security Group ID 是否正确
- Region 配置是否与安全组所在区域一致

**Q: AWS 提示 `DoesNotExist` / `The Instance does not exist`?**
A: 当前 AK/SK + Region 下找不到该实例。请检查:
- `AWS Region` 是否与实例所在区域一致
- `AWS Instance Name` 是否精确匹配(区分大小写)
- AK/SK 是否属于正确账号

**Q: AWS EC2 规则更新失败?**
A: 请检查:
- `AWS EC2 Security Group ID` 是否正确
- AK/SK 是否有 EC2 安全组读写权限
- `AWS Region` 是否与安全组所在区域一致

**Q: Volcengine 提示 `SignatureDoesNotMatch`?**
A: 表示签名校验失败。请检查:
- Volcengine AK/SK 是否填写正确(无前后空格)
- AK/SK 是否已禁用或无效
- 服务器系统时间是否准确(NTP)

**Q: 数据库文件在哪里?**
A: 自动创建在 `instance/config.db`,与可执行文件同级目录。

**Q: 如何修改监听端口?**
A: 编辑 `cmd/server/main.go` 第 47 行,修改 `:9877` 为其他端口。

---

## 📊 日志说明

### 日志级别

- **INFO**: 正常操作记录(IP 检查、规则更新成功)
- **WARNING**: 警告信息(某个 IP 服务不可用、配置跳过)
- **ERROR**: 错误信息(API 调用失败、授权失败)

### 日志示例

```
[INFO] 开始IP检查 (providers=volcengine,aws)...
[INFO] 日志自动清理完成: 已删除 8 条 15 天前日志
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
