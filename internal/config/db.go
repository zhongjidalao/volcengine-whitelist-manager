package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"volcengine-whitelist-manager/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error

	// Ensure instance directory exists
	if _, err := os.Stat("instance"); os.IsNotExist(err) {
		os.Mkdir("instance", 0755)
	}

	dbPath := filepath.Join("instance", "config.db")
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	// Migrate the schema
	DB.AutoMigrate(&models.Settings{}, &models.UpdateLog{})

	// Initialize default settings if empty
	var count int64
	DB.Model(&models.Settings{}).Count(&count)
	if count == 0 {
		DB.Create(&models.Settings{})
	} else {
		var settings models.Settings
		if err := DB.First(&settings).Error; err == nil {
			changed := false

			if strings.TrimSpace(settings.Provider) == "" {
				settings.Provider = "volcengine"
				changed = true
			}
			if strings.TrimSpace(settings.Providers) == "" {
				settings.Providers = settings.Provider
				changed = true
			}
			if strings.TrimSpace(settings.VolcenginePorts) == "" {
				settings.VolcenginePorts = settings.SSHPort
				changed = true
			}
			if strings.TrimSpace(settings.AWSPorts) == "" {
				settings.AWSPorts = settings.SSHPort
				changed = true
			}
			if strings.TrimSpace(settings.AWSEC2Ports) == "" {
				settings.AWSEC2Ports = settings.SSHPort
				changed = true
			}
			if strings.TrimSpace(settings.TencentRegion) == "" {
				settings.TencentRegion = "ap-guangzhou"
				changed = true
			}
			if strings.TrimSpace(settings.TencentPorts) == "" {
				settings.TencentPorts = settings.SSHPort
				changed = true
			}

			// Migrate legacy AWS single-provider settings to dedicated AWS fields.
			if strings.EqualFold(strings.TrimSpace(settings.Provider), "aws") {
				if strings.TrimSpace(settings.AWSAccessKey) == "" {
					settings.AWSAccessKey = settings.AccessKey
					changed = true
				}
				if strings.TrimSpace(settings.AWSSecretKey) == "" {
					settings.AWSSecretKey = settings.SecretKey
					changed = true
				}
				if strings.TrimSpace(settings.AWSRegion) == "" {
					settings.AWSRegion = settings.Region
					changed = true
				}
				if strings.TrimSpace(settings.AWSInstanceName) == "" {
					settings.AWSInstanceName = settings.SecurityGroupID
					changed = true
				}
			}
			if strings.EqualFold(strings.TrimSpace(settings.Provider), "tencent") {
				if strings.TrimSpace(settings.TencentSecretID) == "" {
					settings.TencentSecretID = settings.AccessKey
					changed = true
				}
				if strings.TrimSpace(settings.TencentSecretKey) == "" {
					settings.TencentSecretKey = settings.SecretKey
					changed = true
				}
				if strings.TrimSpace(settings.TencentSecurityGroupID) == "" {
					settings.TencentSecurityGroupID = settings.SecurityGroupID
					changed = true
				}
			}

			if changed {
				DB.Save(&settings)
			}
		}
	}
}

func GetSettings() *models.Settings {
	var settings models.Settings
	DB.First(&settings)
	return &settings
}

func Log(level, message string) {
	if DB != nil {
		DB.Create(&models.UpdateLog{
			Level:   level,
			Message: message,
		})
	}
}

func CleanupOldLogs(days int) (int64, error) {
	if DB == nil || days <= 0 {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	result := DB.Where("timestamp < ?", cutoff).Delete(&models.UpdateLog{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
