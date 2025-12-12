package web

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"volcengine-updater/internal/config"
	"volcengine-updater/internal/models"
	"volcengine-updater/internal/service"

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
		"NextRun":       nextRun,
		"CheckInterval": settings.CheckInterval,
		"SSHPort":       settings.SSHPort,
	})
}

func (h *Handler) GetLogsJSON(c *gin.Context) {
	var logs []models.UpdateLog
	config.DB.Order("timestamp desc").Limit(50).Find(&logs)
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) Index(c *gin.Context) {
	settings := config.GetSettings()
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
		"Settings": settings,
		"Logs":     logs,
		"NextRun":  nextRun,
		"Flash":    flash,
	})
}

func (h *Handler) SettingsPage(c *gin.Context) {
	settings := config.GetSettings()
    
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
		"Settings": settings,
        "CheckIntervalValue": intervalValue,
        "CheckIntervalUnit": intervalUnit,
	})
}

func (h *Handler) SaveSettings(c *gin.Context) {
	var form struct {
		AccessKey       string `form:"access_key"`
		SecretKey       string `form:"secret_key"`
		Region          string `form:"region"`
		SecurityGroupID string `form:"security_group_id"`
		SSHPort         string `form:"ssh_port"`
		CheckInterval   int    `form:"check_interval"`
		IPServices      string `form:"ip_services"`
	}

	if err := c.ShouldBind(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings := config.GetSettings()
	settings.AccessKey = form.AccessKey
	settings.SecretKey = form.SecretKey
	settings.Region = form.Region
	settings.SecurityGroupID = form.SecurityGroupID
	settings.SSHPort = form.SSHPort
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
