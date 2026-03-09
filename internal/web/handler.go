package web

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"volcengine-whitelist-manager/internal/config"
	"volcengine-whitelist-manager/internal/models"
	"volcengine-whitelist-manager/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type Handler struct {
	Cron  *cron.Cron
	JobID cron.EntryID
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.Index)
	r.GET("/settings", h.SettingsPage)
	r.POST("/settings", h.SaveSettings)
	r.POST("/run_now", h.RunNow)
	r.POST("/logs/clear", h.ClearLogs)
	r.GET("/logs", h.LogsPage)
	r.GET("/api/logs", h.GetLogsJSON)
	r.GET("/api/status", h.GetStatusJSON)
}

func (h *Handler) ClearLogs(c *gin.Context) {
	config.DB.Exec("DELETE FROM update_logs")
	c.Redirect(http.StatusFound, "/logs?flash=日志已清空")
}

func (h *Handler) GetStatusJSON(c *gin.Context) {
	settings := config.GetSettings()
	providers := resolveProviders(settings)

	var nextRun string
	if h.JobID != 0 {
		entry := h.Cron.Entry(h.JobID)
		if !entry.Next.IsZero() {
			nextRun = entry.Next.Format("2006-01-02 15:04:05")
		} else {
			nextRun = "运行中..."
		}
	} else {
		nextRun = "未调度"
	}

	c.JSON(http.StatusOK, gin.H{
		"NextRun":         nextRun,
		"CheckInterval":   settings.CheckInterval,
		"SSHPort":         settings.SSHPort,
		"VolcenginePorts": firstNonEmpty(settings.VolcenginePorts, settings.SSHPort),
		"AWSPorts":        firstNonEmpty(settings.AWSPorts, settings.SSHPort),
		"AWSEC2Ports":     firstNonEmpty(settings.AWSEC2Ports, settings.SSHPort),
		"TencentPorts":    firstNonEmpty(settings.TencentPorts, settings.SSHPort),
		"Provider":        settings.Provider,
		"Providers":       providers,
		"VolcEnabled":     hasProvider(providers, "volcengine"),
		"AWSEnabled":      hasProvider(providers, "aws"),
		"AWSEC2Enabled":   hasProvider(providers, "aws-ec2"),
		"TencentEnabled":  hasProvider(providers, "tencent"),
	})
}

func (h *Handler) GetLogsJSON(c *gin.Context) {
	var logs []models.UpdateLog
	config.DB.Order("timestamp desc").Limit(50).Find(&logs)
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) Index(c *gin.Context) {
	settings := config.GetSettings()
	providers := resolveProviders(settings)
	var logs []models.UpdateLog
	config.DB.Order("timestamp desc").Limit(10).Find(&logs)

	var nextRun string
	if h.JobID != 0 {
		entry := h.Cron.Entry(h.JobID)
		if !entry.Next.IsZero() {
			nextRun = entry.Next.Format("2006-01-02 15:04:05")
		} else {
			nextRun = "Running..."
		}
	} else {
		nextRun = "Not Scheduled"
	}

	flash := c.Query("flash")

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Settings":       settings,
		"Logs":           logs,
		"NextRun":        nextRun,
		"Flash":          flash,
		"VolcEnabled":    hasProvider(providers, "volcengine"),
		"AWSEnabled":     hasProvider(providers, "aws"),
		"AWSEC2Enabled":  hasProvider(providers, "aws-ec2"),
		"TencentEnabled": hasProvider(providers, "tencent"),
	})
}

func (h *Handler) SettingsPage(c *gin.Context) {
	settings := config.GetSettings()
	if strings.TrimSpace(settings.Provider) == "" {
		settings.Provider = "volcengine"
	}
	if strings.TrimSpace(settings.Providers) == "" {
		settings.Providers = settings.Provider
	}
	if strings.TrimSpace(settings.VolcenginePorts) == "" {
		settings.VolcenginePorts = settings.SSHPort
	}
	if strings.TrimSpace(settings.AWSPorts) == "" {
		settings.AWSPorts = settings.SSHPort
	}
	if strings.TrimSpace(settings.AWSEC2Ports) == "" {
		settings.AWSEC2Ports = settings.SSHPort
	}
	if strings.TrimSpace(settings.TencentPorts) == "" {
		settings.TencentPorts = settings.SSHPort
	}

	// Convert CheckInterval (seconds) to human-readable form
	intervalValue := settings.CheckInterval
	intervalUnit := "seconds" // Default unit

	if intervalValue >= 3600 && intervalValue%3600 == 0 { // Check for full hours
		intervalValue /= 3600
		intervalUnit = "hours"
	} else if intervalValue >= 60 && intervalValue%60 == 0 { // Check for full minutes
		intervalValue /= 60
		intervalUnit = "minutes"
	}

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"Settings":           settings,
		"CheckIntervalValue": intervalValue,
		"CheckIntervalUnit":  intervalUnit,
		"VolcEnabled":        hasProvider(settings.Providers, "volcengine"),
		"AWSEnabled":         hasProvider(settings.Providers, "aws"),
		"AWSEC2Enabled":      hasProvider(settings.Providers, "aws-ec2"),
		"TencentEnabled":     hasProvider(settings.Providers, "tencent"),
	})
}

func (h *Handler) SaveSettings(c *gin.Context) {
	var form struct {
		Providers               []string `form:"providers"`
		VolcengineAccessKey     string   `form:"volcengine_access_key"`
		VolcengineSecretKey     string   `form:"volcengine_secret_key"`
		VolcengineRegion        string   `form:"volcengine_region"`
		VolcengineSecurityGroup string   `form:"volcengine_security_group_id"`
		AWSAccessKey            string   `form:"aws_access_key"`
		AWSSecretKey            string   `form:"aws_secret_key"`
		AWSRegion               string   `form:"aws_region"`
		AWSInstanceName         string   `form:"aws_instance_name"`
		AWSEC2SecurityGroupID   string   `form:"aws_ec2_security_group_id"`
		TencentSecretID         string   `form:"tencent_secret_id"`
		TencentSecretKey        string   `form:"tencent_secret_key"`
		TencentRegion           string   `form:"tencent_region"`
		TencentSecurityGroupID  string   `form:"tencent_security_group_id"`
		VolcenginePorts         string   `form:"volcengine_ports"`
		AWSPorts                string   `form:"aws_ports"`
		AWSEC2Ports             string   `form:"aws_ec2_ports"`
		TencentPorts            string   `form:"tencent_ports"`
		CheckInterval           int      `form:"check_interval"`
		IPServices              string   `form:"ip_services"`
	}

	if err := c.ShouldBind(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings := config.GetSettings()
	providers := normalizeProvidersFromForm(form.Providers)
	settings.Providers = providers
	switch providers {
	case "aws":
		settings.Provider = "aws"
	case "tencent":
		settings.Provider = "tencent"
	case "":
		settings.Provider = ""
	default:
		settings.Provider = "volcengine"
	}

	settings.AccessKey = form.VolcengineAccessKey
	settings.SecretKey = form.VolcengineSecretKey
	settings.Region = form.VolcengineRegion
	settings.SecurityGroupID = form.VolcengineSecurityGroup
	settings.AWSAccessKey = form.AWSAccessKey
	settings.AWSSecretKey = form.AWSSecretKey
	settings.AWSRegion = form.AWSRegion
	settings.AWSInstanceName = form.AWSInstanceName
	settings.AWSEC2SecurityGroupID = form.AWSEC2SecurityGroupID
	settings.TencentSecretID = form.TencentSecretID
	settings.TencentSecretKey = form.TencentSecretKey
	settings.TencentRegion = form.TencentRegion
	settings.TencentSecurityGroupID = form.TencentSecurityGroupID
	settings.VolcenginePorts = form.VolcenginePorts
	settings.AWSPorts = form.AWSPorts
	settings.AWSEC2Ports = form.AWSEC2Ports
	settings.TencentPorts = form.TencentPorts
	settings.SSHPort = firstNonEmpty(form.VolcenginePorts, form.AWSPorts, form.AWSEC2Ports, form.TencentPorts)
	settings.CheckInterval = form.CheckInterval
	settings.IPServices = form.IPServices

	config.DB.Save(settings)

	// Update Job
	if h.JobID != 0 {
		h.Cron.Remove(h.JobID)
	}

	id, _ := h.Cron.AddFunc(fmt.Sprintf("@every %ds", settings.CheckInterval), service.CheckAndUpdate)
	h.JobID = id

	c.Redirect(http.StatusFound, "/?flash=设置已保存")
}

func normalizeProvidersFromForm(rawProviders []string) string {
	seen := make(map[string]struct{}, len(rawProviders))
	for _, provider := range rawProviders {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider != "volcengine" && provider != "aws" && provider != "aws-ec2" && provider != "tencent" {
			continue
		}
		seen[provider] = struct{}{}
	}

	ordered := make([]string, 0, 4)
	if _, ok := seen["volcengine"]; ok {
		ordered = append(ordered, "volcengine")
	}
	if _, ok := seen["aws"]; ok {
		ordered = append(ordered, "aws")
	}
	if _, ok := seen["aws-ec2"]; ok {
		ordered = append(ordered, "aws-ec2")
	}
	if _, ok := seen["tencent"]; ok {
		ordered = append(ordered, "tencent")
	}
	return strings.Join(ordered, ",")
}

func resolveProviders(settings *models.Settings) string {
	providers := strings.TrimSpace(settings.Providers)
	if providers == "" {
		providers = strings.TrimSpace(settings.Provider)
	}
	if providers == "" {
		providers = "volcengine"
	}
	return providers
}

func hasProvider(providersCSV, target string) bool {
	for _, provider := range strings.Split(providersCSV, ",") {
		if strings.EqualFold(strings.TrimSpace(provider), target) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) RunNow(c *gin.Context) {
	go service.CheckAndUpdate()
	c.Redirect(http.StatusFound, "/?flash=已触发立即更新任务")
}

func (h *Handler) LogsPage(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	pageSize := 20

	var total int64
	config.DB.Model(&models.UpdateLog{}).Count(&total)

	var logs []models.UpdateLog
	offset := (page - 1) * pageSize
	config.DB.Order("timestamp desc").Limit(pageSize).Offset(offset).Find(&logs)

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// Helper for pagination
	hasPrev := page > 1
	hasNext := page < totalPages

	c.HTML(http.StatusOK, "logs.html", gin.H{
		"Logs":       logs,
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    hasPrev,
		"HasNext":    hasNext,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
	})
}
