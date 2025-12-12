package service

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"volcengine-updater/internal/config"
	"volcengine-updater/internal/models"

	"github.com/volcengine/volcengine-go-sdk/service/vpc"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// CheckAndUpdate is the main entry point for the scheduled task
func CheckAndUpdate() {
	settings := config.GetSettings()
	if settings.AccessKey == "" || settings.SecretKey == "" || settings.SecurityGroupID == "" {
		config.Log("WARNING", "任务跳过: 配置不完整 (AK/SK/SG_ID 缺失)")
		return
	}

	config.Log("INFO", "开始IP检查...")

	currentIP := getCurrentIP(settings.IPServices)
	if currentIP == "" {
		config.Log("ERROR", "无法获取当前公网IP，跳过检查")
		return
	}

	updateSecurityGroup(settings, currentIP)
}

func getCurrentIP(servicesStr string) string {
	services := strings.Split(servicesStr, "\n")
	client := &http.Client{Timeout: 5 * time.Second}

	for _, url := range services {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		resp, err := client.Get(url)
		if err != nil {
			config.Log("WARNING", fmt.Sprintf("从 %s 获取IP失败: %v", url, err))
			continue
		}
		

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			ip := strings.TrimSpace(string(body))
			if ip != "" {
				config.Log("INFO", fmt.Sprintf("当前公网IP: %s (来源: %s)", ip, url))
				return ip
			}
		} else {
			resp.Body.Close()
		}
	}
	return ""
}

func updateSecurityGroup(settings *models.Settings, currentIP string) {
	conf := volcengine.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(settings.AccessKey, settings.SecretKey, "")).
		WithRegion(settings.Region)

	sess, err := session.NewSession(conf)
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("创建会话失败: %v", err))
		return
	}

	vpcClient := vpc.New(sess)

	// Parse Ports
	portsStr := strings.Split(settings.SSHPort, ",")
	var ports []int
	for _, p := range portsStr {
		p = strings.TrimSpace(p)
		// Check for range like "8000-8005" - simpler to just support single ports for now, 
		// or treat ranges? User asked for comma separated.
		if val, err := strconv.Atoi(p); err == nil && val > 0 && val <= 65535 {
			ports = append(ports, val)
		}
	}

	if len(ports) == 0 {
		config.Log("WARNING", "未配置有效的SSH端口 (请使用逗号分隔，例如: 22,8080)")
		return
	}

	// Get current rules
	input := &vpc.DescribeSecurityGroupAttributesInput{
		SecurityGroupId: volcengine.String(settings.SecurityGroupID),
	}

	output, err := vpcClient.DescribeSecurityGroupAttributes(input)
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("获取安全组属性失败: %v", err))
		return
	}

	for _, targetPort := range ports {
		var existingRule *vpc.PermissionForDescribeSecurityGroupAttributesOutput
		description := fmt.Sprintf("SSH访问(端口%d) - Go脚本自动更新", targetPort)

		// Find existing SSH rule for THIS port
		for _, perm := range output.Permissions {
			if volcengine.StringValue(perm.Direction) == "ingress" &&
				(strings.EqualFold(volcengine.StringValue(perm.Protocol), "tcp") || strings.EqualFold(volcengine.StringValue(perm.Protocol), "all")) &&
				int(volcengine.Int64Value(perm.PortStart)) == targetPort &&
				int(volcengine.Int64Value(perm.PortEnd)) == targetPort {
				existingRule = perm
				if desc := volcengine.StringValue(perm.Description); desc != "" {
					description = desc
				}
				break
			}
		}

		if existingRule != nil {
			currentCidr := volcengine.StringValue(existingRule.CidrIp)
			existingIP := strings.Split(currentCidr, "/")[0]
			
			if existingIP == currentIP {
				config.Log("INFO", fmt.Sprintf("端口 %d: IP未变 (%s)，无需更新", targetPort, existingIP))
				continue
			}

			// Revoke old rule
			config.Log("INFO", fmt.Sprintf("端口 %d: 撤销旧规则 %s", targetPort, currentCidr))
			_, err := vpcClient.RevokeSecurityGroupIngress(&vpc.RevokeSecurityGroupIngressInput{
				SecurityGroupId: volcengine.String(settings.SecurityGroupID),
				Protocol:        existingRule.Protocol,
				PortStart:       volcengine.Int64(int64(targetPort)),
				PortEnd:         volcengine.Int64(int64(targetPort)),
				CidrIp:          existingRule.CidrIp,
				Policy:          existingRule.Policy,
			})
			if err != nil {
				config.Log("WARNING", fmt.Sprintf("端口 %d: 撤销失败: %v", targetPort, err))
			}
		} else {
			config.Log("INFO", fmt.Sprintf("端口 %d: 未找到现有规则，将添加新规则", targetPort))
		}

		// Authorize new rule
		newCidr := fmt.Sprintf("%s/32", currentIP)
		config.Log("INFO", fmt.Sprintf("端口 %d: 添加新规则 %s", targetPort, newCidr))
		
		_, err = vpcClient.AuthorizeSecurityGroupIngress(&vpc.AuthorizeSecurityGroupIngressInput{
			SecurityGroupId: volcengine.String(settings.SecurityGroupID),
			Protocol:        volcengine.String("TCP"),
			PortStart:       volcengine.Int64(int64(targetPort)),
			PortEnd:         volcengine.Int64(int64(targetPort)),
			CidrIp:          volcengine.String(newCidr),
			Policy:          volcengine.String("accept"),
			Priority:        volcengine.Int64(1),
			Description:     volcengine.String(description),
		})

		if err != nil {
			config.Log("ERROR", fmt.Sprintf("端口 %d: 授权失败: %v", targetPort, err))
		} else {
			config.Log("INFO", fmt.Sprintf("✓ 端口 %d: 已更新允许 %s", targetPort, newCidr))
		}
	}
}