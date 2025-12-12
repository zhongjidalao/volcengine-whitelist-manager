package models

import (
	"time"
)

type Settings struct {
	ID              uint   `gorm:"primaryKey"`
	AccessKey       string `gorm:"size:100"`
	SecretKey       string `gorm:"size:100"`
	Region          string `gorm:"size:50;default:'cn-beijing'"`
	SecurityGroupID string `gorm:"size:50"`
	SSHPort         string `gorm:"size:50;default:'22'"`
	CheckInterval   int    `gorm:"default:900"` // Seconds
	IPServices      string `gorm:"type:text;default:'https://myip.ipip.net\nhttps://ddns.oray.com/checkip\nhttps://ip.3322.net\nhttps://v4.yinghualuo.cn/bejson'"`
}

type UpdateLog struct {
	ID        uint      `gorm:"primaryKey"`
	Timestamp time.Time `gorm:"autoCreateTime"`
	Level     string    `gorm:"size:20"` // INFO, ERROR, WARNING
	Message   string    `gorm:"type:text"`
}
