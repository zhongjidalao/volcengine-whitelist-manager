package models

import (
	"time"
)

type Settings struct {
	ID                     uint   `gorm:"primaryKey"`
	Provider               string `gorm:"size:20;default:'volcengine'"` // Legacy single provider field
	Providers              string `gorm:"size:50;default:'volcengine'"` // Comma-separated providers
	AccessKey              string `gorm:"size:100"`                     // Volcengine Access Key
	SecretKey              string `gorm:"size:100"`                     // Volcengine Secret Key
	Region                 string `gorm:"size:50;default:'cn-beijing'"` // Volcengine Region
	SecurityGroupID        string `gorm:"size:255"`                     // Volcengine Security Group ID
	VolcenginePorts        string `gorm:"size:50;default:'22'"`
	AWSAccessKey           string `gorm:"size:100"` // AWS Access Key
	AWSSecretKey           string `gorm:"size:100"` // AWS Secret Key
	AWSRegion              string `gorm:"size:50;default:'ap-southeast-1'"`
	AWSInstanceName        string `gorm:"size:255"`
	AWSPorts               string `gorm:"size:50;default:'22'"`
	AWSEC2SecurityGroupID  string `gorm:"size:255"`
	AWSEC2Ports            string `gorm:"size:50;default:'22'"`
	TencentSecretID        string `gorm:"size:100"` // Tencent Cloud SecretId
	TencentSecretKey       string `gorm:"size:100"` // Tencent Cloud SecretKey
	TencentRegion          string `gorm:"size:50;default:'ap-guangzhou'"`
	TencentSecurityGroupID string `gorm:"size:255"`
	TencentPorts           string `gorm:"size:50;default:'22'"`
	AliyunAccessKey        string `gorm:"size:100"` // Alibaba Cloud AccessKey ID
	AliyunSecretKey        string `gorm:"size:100"` // Alibaba Cloud AccessKey Secret
	AliyunRegion           string `gorm:"size:50;default:'cn-hangzhou'"`
	AliyunSecurityGroupID  string `gorm:"size:255"`
	AliyunPorts            string `gorm:"size:50;default:'22'"`
	SSHPort                string `gorm:"size:50;default:'22'"`
	CheckInterval          int    `gorm:"default:900"` // Seconds
	IPServices             string `gorm:"type:text;default:'https://myip.ipip.net\nhttps://ddns.oray.com/checkip\nhttps://ip.3322.net\nhttps://v4.yinghualuo.cn/bejson'"`
}

type UpdateLog struct {
	ID        uint      `gorm:"primaryKey"`
	Timestamp time.Time `gorm:"autoCreateTime"`
	Level     string    `gorm:"size:20"` // INFO, ERROR, WARNING
	Message   string    `gorm:"type:text"`
}
