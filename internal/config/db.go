package config

import (
	"os"
	"path/filepath"
	"volcengine-whitelist-manager/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // Use pure Go SQLite implementation
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
