package service

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"volcengine-whitelist-manager/internal/config"
	"volcengine-whitelist-manager/internal/models"

	aliyunOpenAPI "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	aliyunECS "github.com/alibabacloud-go/ecs-20140526/v7/client"
	awsSDK "github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	awsec2 "github.com/aws/aws-sdk-go/service/ec2"
	awslightsail "github.com/aws/aws-sdk-go/service/lightsail"
	tencentCommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentProfile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentVPC "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"github.com/volcengine/volcengine-go-sdk/service/vpc"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	volcCredentials "github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	volcSession "github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

const (
	providerVolcengine = "volcengine"
	providerAWS        = "aws"
	providerAWSEC2     = "aws-ec2"
	providerTencent    = "tencent"
	providerAliyun     = "aliyun"
)

// CheckAndUpdate is the main entry point for the scheduled task
func CheckAndUpdate() {
	if deletedCount, err := config.CleanupOldLogs(15); err != nil {
		config.Log("WARNING", fmt.Sprintf("日志自动清理失败: %v", err))
	} else if deletedCount > 0 {
		config.Log("INFO", fmt.Sprintf("日志自动清理完成: 已删除 %d 条 15 天前日志", deletedCount))
	}

	settings := config.GetSettings()
	providers := normalizeProviders(settings.Providers, settings.Provider)
	if len(providers) == 0 {
		config.Log("WARNING", "任务跳过: 未选择任何云供应商")
		return
	}

	config.Log("INFO", fmt.Sprintf("开始IP检查 (providers=%s)...", strings.Join(providers, ",")))

	currentIP := getCurrentIP(settings.IPServices)
	if currentIP == "" {
		config.Log("ERROR", "无法获取当前公网IP，跳过检查")
		return
	}

	for _, provider := range providers {
		if err := validateSettings(settings, provider); err != nil {
			config.Log("WARNING", err.Error())
			continue
		}
		ports := parsePorts(getPortsByProvider(settings, provider))
		if len(ports) == 0 {
			config.Log("WARNING", fmt.Sprintf("provider=%s 跳过: 未配置有效端口 (请使用逗号分隔，例如: 22,8080)", provider))
			continue
		}

		switch provider {
		case providerAWS:
			updateAWSLightsailFirewall(settings, currentIP, ports)
		case providerAWSEC2:
			updateAWSEC2SecurityGroup(settings, currentIP, ports)
		case providerTencent:
			updateTencentSecurityGroup(settings, currentIP, ports)
		case providerAliyun:
			updateAliyunSecurityGroup(settings, currentIP, ports)
		default:
			updateVolcengineSecurityGroup(settings, currentIP, ports)
		}
	}
}

func normalizeProviders(providersStr, legacyProvider string) []string {
	raw := strings.TrimSpace(providersStr)
	if raw == "" {
		raw = strings.TrimSpace(legacyProvider)
	}
	if raw == "" {
		raw = providerVolcengine
	}

	providers := make([]string, 0, 5)
	seen := make(map[string]struct{}, 5)
	for _, item := range strings.Split(raw, ",") {
		provider, ok := normalizeProvider(item)
		if !ok {
			continue
		}
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		providers = append(providers, provider)
	}

	return providers
}

func normalizeProvider(provider string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providerVolcengine:
		return providerVolcengine, true
	case providerAWS:
		return providerAWS, true
	case providerAWSEC2:
		return providerAWSEC2, true
	case providerTencent:
		return providerTencent, true
	case providerAliyun:
		return providerAliyun, true
	default:
		return "", false
	}
}

func validateSettings(settings *models.Settings, provider string) error {
	var missing []string
	switch provider {
	case providerAWS:
		if strings.TrimSpace(settings.AWSAccessKey) == "" {
			missing = append(missing, "AWS_AK")
		}
		if strings.TrimSpace(settings.AWSSecretKey) == "" {
			missing = append(missing, "AWS_SK")
		}
		if strings.TrimSpace(settings.AWSRegion) == "" {
			missing = append(missing, "AWS_Region")
		}
		if strings.TrimSpace(settings.AWSInstanceName) == "" {
			missing = append(missing, "AWS_InstanceName")
		}
	case providerAWSEC2:
		if strings.TrimSpace(settings.AWSAccessKey) == "" {
			missing = append(missing, "AWS_AK")
		}
		if strings.TrimSpace(settings.AWSSecretKey) == "" {
			missing = append(missing, "AWS_SK")
		}
		if strings.TrimSpace(settings.AWSRegion) == "" {
			missing = append(missing, "AWS_Region")
		}
		if strings.TrimSpace(settings.AWSEC2SecurityGroupID) == "" {
			missing = append(missing, "AWS_EC2_SG_ID")
		}
	case providerTencent:
		if strings.TrimSpace(settings.TencentSecretID) == "" {
			missing = append(missing, "Tencent_SecretID")
		}
		if strings.TrimSpace(settings.TencentSecretKey) == "" {
			missing = append(missing, "Tencent_SecretKey")
		}
		if strings.TrimSpace(settings.TencentRegion) == "" {
			missing = append(missing, "Tencent_Region")
		}
		if strings.TrimSpace(settings.TencentSecurityGroupID) == "" {
			missing = append(missing, "Tencent_SG_ID")
		}
	case providerAliyun:
		if strings.TrimSpace(settings.AliyunAccessKey) == "" {
			missing = append(missing, "Aliyun_AK")
		}
		if strings.TrimSpace(settings.AliyunSecretKey) == "" {
			missing = append(missing, "Aliyun_SK")
		}
		if strings.TrimSpace(settings.AliyunRegion) == "" {
			missing = append(missing, "Aliyun_Region")
		}
		if strings.TrimSpace(settings.AliyunSecurityGroupID) == "" {
			missing = append(missing, "Aliyun_SG_ID")
		}
	default:
		if strings.TrimSpace(settings.AccessKey) == "" {
			missing = append(missing, "Volc_AK")
		}
		if strings.TrimSpace(settings.SecretKey) == "" {
			missing = append(missing, "Volc_SK")
		}
		if strings.TrimSpace(settings.Region) == "" {
			missing = append(missing, "Volc_Region")
		}
		if strings.TrimSpace(settings.SecurityGroupID) == "" {
			missing = append(missing, "Volc_SG_ID")
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("provider=%s 跳过: 配置不完整 (%s 缺失)", provider, strings.Join(missing, "/"))
	}

	return nil
}

// extractIP extracts IPv4 address from text using regex
func extractIP(text string) string {
	// IPv4 正则表达式模式
	ipPattern := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	match := ipPattern.FindString(text)
	if match != "" {
		// 验证 IP 地址的每个部分是否在 0-255 范围内
		parts := strings.Split(match, ".")
		valid := true
		for _, part := range parts {
			if num, err := strconv.Atoi(part); err != nil || num < 0 || num > 255 {
				valid = false
				break
			}
		}
		if valid {
			return match
		}
	}
	return ""
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
			responseText := strings.TrimSpace(string(body))

			// 尝试从响应文本中提取 IP 地址
			ip := extractIP(responseText)

			// 如果提取失败，尝试直接使用响应内容（兼容纯 IP 响应）
			if ip == "" && net.ParseIP(responseText) != nil {
				ip = responseText
			}

			if ip != "" {
				config.Log("INFO", fmt.Sprintf("当前公网IP: %s (来源: %s)", ip, url))
				return ip
			} else {
				config.Log("WARNING", fmt.Sprintf("从 %s 无法解析IP地址，响应内容: %s", url, responseText))
			}
		} else {
			resp.Body.Close()
		}
	}
	return ""
}

func parsePorts(portsInput string) []int {
	portsStr := strings.Split(portsInput, ",")
	ports := make([]int, 0, len(portsStr))
	seen := make(map[int]struct{}, len(portsStr))

	for _, p := range portsStr {
		p = strings.TrimSpace(p)
		if val, err := strconv.Atoi(p); err == nil && val > 0 && val <= 65535 {
			if _, ok := seen[val]; ok {
				continue
			}
			seen[val] = struct{}{}
			ports = append(ports, val)
		}
	}

	return ports
}

func getPortsByProvider(settings *models.Settings, provider string) string {
	switch provider {
	case providerAWS:
		if ports := strings.TrimSpace(settings.AWSPorts); ports != "" {
			return ports
		}
	case providerAWSEC2:
		if ports := strings.TrimSpace(settings.AWSEC2Ports); ports != "" {
			return ports
		}
	case providerTencent:
		if ports := strings.TrimSpace(settings.TencentPorts); ports != "" {
			return ports
		}
	case providerAliyun:
		if ports := strings.TrimSpace(settings.AliyunPorts); ports != "" {
			return ports
		}
	default:
		if ports := strings.TrimSpace(settings.VolcenginePorts); ports != "" {
			return ports
		}
	}

	return strings.TrimSpace(settings.SSHPort)
}

func updateVolcengineSecurityGroup(settings *models.Settings, currentIP string, ports []int) {
	conf := volcengine.NewConfig().
		WithCredentials(volcCredentials.NewStaticCredentials(settings.AccessKey, settings.SecretKey, "")).
		WithRegion(settings.Region)

	sess, err := volcSession.NewSession(conf)
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("创建会话失败: %v", err))
		return
	}

	vpcClient := vpc.New(sess)

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
		description := fmt.Sprintf("白名单访问(端口%d) - Go脚本自动更新", targetPort)

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

func updateAWSLightsailFirewall(settings *models.Settings, currentIP string, ports []int) {
	region := strings.TrimSpace(settings.AWSRegion)
	normalizedRegion, regionChanged := normalizeAWSRegion(region)
	if regionChanged {
		config.Log("WARNING", fmt.Sprintf("provider=aws: 区域使用了可用区格式 (%s)，已自动纠正为 %s", region, normalizedRegion))
	}

	sess, err := awsSession.NewSession(&awsSDK.Config{
		Region:      awsSDK.String(normalizedRegion),
		Credentials: awsCredentials.NewStaticCredentials(settings.AWSAccessKey, settings.AWSSecretKey, ""),
	})
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("provider=aws: 创建AWS会话失败: %v", err))
		return
	}

	client := awslightsail.New(sess)
	instanceName := strings.TrimSpace(settings.AWSInstanceName)

	output, err := client.GetInstancePortStates(&awslightsail.GetInstancePortStatesInput{
		InstanceName: awsSDK.String(instanceName),
	})
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("provider=aws: 获取Lightsail端口状态失败: %v", err))
		return
	}

	newCidr := fmt.Sprintf("%s/32", currentIP)

	for _, targetPort := range ports {
		matchedStates := findManagedLightsailStates(output.PortStates, targetPort)
		if isLightsailPortSynced(matchedStates, targetPort, newCidr) {
			config.Log("INFO", fmt.Sprintf("provider=aws 端口 %d: IP未变 (%s)，无需更新", targetPort, currentIP))
			continue
		}

		closedProtocols := make(map[string]struct{}, len(matchedStates))
		for _, state := range matchedStates {
			protocol := strings.ToLower(awsSDK.StringValue(state.Protocol))
			if protocol == "" {
				continue
			}
			if _, ok := closedProtocols[protocol]; ok {
				continue
			}

			config.Log("INFO", fmt.Sprintf("provider=aws 端口 %d: 关闭旧规则(protocol=%s)", targetPort, protocol))
			_, err = client.CloseInstancePublicPorts(&awslightsail.CloseInstancePublicPortsInput{
				InstanceName: awsSDK.String(instanceName),
				PortInfo: &awslightsail.PortInfo{
					FromPort: awsSDK.Int64(int64(targetPort)),
					ToPort:   awsSDK.Int64(int64(targetPort)),
					Protocol: awsSDK.String(protocol),
				},
			})
			if err != nil {
				config.Log("WARNING", fmt.Sprintf("provider=aws 端口 %d: 关闭旧规则失败: %v", targetPort, err))
			}
			closedProtocols[protocol] = struct{}{}
		}

		if len(matchedStates) == 0 {
			config.Log("INFO", fmt.Sprintf("provider=aws 端口 %d: 未找到现有规则，将添加新规则", targetPort))
		}

		config.Log("INFO", fmt.Sprintf("provider=aws 端口 %d: 添加新规则 %s", targetPort, newCidr))
		_, err = client.OpenInstancePublicPorts(&awslightsail.OpenInstancePublicPortsInput{
			InstanceName: awsSDK.String(instanceName),
			PortInfo: &awslightsail.PortInfo{
				FromPort: awsSDK.Int64(int64(targetPort)),
				ToPort:   awsSDK.Int64(int64(targetPort)),
				Protocol: awsSDK.String(awslightsail.NetworkProtocolTcp),
				Cidrs:    []*string{awsSDK.String(newCidr)},
			},
		})
		if err != nil {
			config.Log("ERROR", fmt.Sprintf("provider=aws 端口 %d: 授权失败: %v", targetPort, err))
		} else {
			config.Log("INFO", fmt.Sprintf("✓ provider=aws 端口 %d: 已更新允许 %s", targetPort, newCidr))
		}
	}
}

func updateAWSEC2SecurityGroup(settings *models.Settings, currentIP string, ports []int) {
	region := strings.TrimSpace(settings.AWSRegion)
	normalizedRegion, regionChanged := normalizeAWSRegion(region)
	if regionChanged {
		config.Log("WARNING", fmt.Sprintf("provider=aws-ec2: 区域使用了可用区格式 (%s)，已自动纠正为 %s", region, normalizedRegion))
	}

	sess, err := awsSession.NewSession(&awsSDK.Config{
		Region:      awsSDK.String(normalizedRegion),
		Credentials: awsCredentials.NewStaticCredentials(settings.AWSAccessKey, settings.AWSSecretKey, ""),
	})
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("provider=aws-ec2: 创建AWS会话失败: %v", err))
		return
	}

	client := awsec2.New(sess)
	securityGroupID := strings.TrimSpace(settings.AWSEC2SecurityGroupID)

	rulesOutput, err := client.DescribeSecurityGroupRules(&awsec2.DescribeSecurityGroupRulesInput{
		Filters: []*awsec2.Filter{
			{
				Name:   awsSDK.String("group-id"),
				Values: []*string{awsSDK.String(securityGroupID)},
			},
		},
	})
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("provider=aws-ec2: 获取安全组规则失败: %v", err))
		return
	}

	newCidr := fmt.Sprintf("%s/32", currentIP)
	for _, targetPort := range ports {
		matchedRules := findManagedAWSEC2Rules(rulesOutput.SecurityGroupRules, targetPort)
		if isAWSEC2PortSynced(matchedRules, targetPort, newCidr) {
			config.Log("INFO", fmt.Sprintf("provider=aws-ec2 端口 %d: IP未变 (%s)，无需更新", targetPort, currentIP))
			continue
		}

		ruleIDs := make([]*string, 0, len(matchedRules))
		for _, rule := range matchedRules {
			if rule.SecurityGroupRuleId == nil {
				continue
			}
			ruleIDs = append(ruleIDs, rule.SecurityGroupRuleId)
		}

		if len(ruleIDs) > 0 {
			config.Log("INFO", fmt.Sprintf("provider=aws-ec2 端口 %d: 撤销旧规则 %d 条", targetPort, len(ruleIDs)))
			_, err = client.RevokeSecurityGroupIngress(&awsec2.RevokeSecurityGroupIngressInput{
				GroupId:              awsSDK.String(securityGroupID),
				SecurityGroupRuleIds: ruleIDs,
			})
			if err != nil {
				config.Log("WARNING", fmt.Sprintf("provider=aws-ec2 端口 %d: 撤销旧规则失败: %v", targetPort, err))
			}
		} else {
			config.Log("INFO", fmt.Sprintf("provider=aws-ec2 端口 %d: 未找到现有规则，将添加新规则", targetPort))
		}

		config.Log("INFO", fmt.Sprintf("provider=aws-ec2 端口 %d: 添加新规则 %s", targetPort, newCidr))
		_, err = client.AuthorizeSecurityGroupIngress(&awsec2.AuthorizeSecurityGroupIngressInput{
			GroupId: awsSDK.String(securityGroupID),
			IpPermissions: []*awsec2.IpPermission{
				{
					IpProtocol: awsSDK.String("tcp"),
					FromPort:   awsSDK.Int64(int64(targetPort)),
					ToPort:     awsSDK.Int64(int64(targetPort)),
					IpRanges: []*awsec2.IpRange{
						{
							CidrIp:      awsSDK.String(newCidr),
							Description: awsSDK.String(fmt.Sprintf("白名单访问(端口%d) - Go脚本自动更新", targetPort)),
						},
					},
				},
			},
		})
		if err != nil {
			config.Log("ERROR", fmt.Sprintf("provider=aws-ec2 端口 %d: 授权失败: %v", targetPort, err))
		} else {
			config.Log("INFO", fmt.Sprintf("✓ provider=aws-ec2 端口 %d: 已更新允许 %s", targetPort, newCidr))
		}
	}
}

func updateTencentSecurityGroup(settings *models.Settings, currentIP string, ports []int) {
	client, err := tencentVPC.NewClient(
		tencentCommon.NewCredential(
			strings.TrimSpace(settings.TencentSecretID),
			strings.TrimSpace(settings.TencentSecretKey),
		),
		strings.TrimSpace(settings.TencentRegion),
		tencentProfile.NewClientProfile(),
	)
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("provider=tencent: 创建腾讯云会话失败: %v", err))
		return
	}

	securityGroupID := strings.TrimSpace(settings.TencentSecurityGroupID)
	newCidr := fmt.Sprintf("%s/32", currentIP)

	for _, targetPort := range ports {
		ingressRules, err := describeTencentIngressPolicies(client, securityGroupID)
		if err != nil {
			config.Log("ERROR", fmt.Sprintf("provider=tencent: 获取安全组规则失败: %v", err))
			return
		}

		matchedRules := findManagedTencentIngressRules(ingressRules, targetPort)
		if isTencentPortSynced(matchedRules, targetPort, newCidr) {
			config.Log("INFO", fmt.Sprintf("provider=tencent 端口 %d: IP未变 (%s)，无需更新", targetPort, currentIP))
			continue
		}

		policyIndexes := collectTencentPolicyIndexes(matchedRules)
		if len(policyIndexes) > 0 {
			sort.Slice(policyIndexes, func(i, j int) bool {
				return policyIndexes[i] > policyIndexes[j]
			})

			for _, policyIndex := range policyIndexes {
				config.Log("INFO", fmt.Sprintf("provider=tencent 端口 %d: 撤销旧规则 index=%d", targetPort, policyIndex))
				deleteReq := tencentVPC.NewDeleteSecurityGroupPoliciesRequest()
				deleteReq.SecurityGroupId = tencentCommon.StringPtr(securityGroupID)
				deleteReq.SecurityGroupPolicySet = &tencentVPC.SecurityGroupPolicySet{
					Ingress: []*tencentVPC.SecurityGroupPolicy{
						{
							PolicyIndex: tencentCommon.Int64Ptr(policyIndex),
						},
					},
				}
				if _, err = client.DeleteSecurityGroupPolicies(deleteReq); err != nil {
					config.Log("WARNING", fmt.Sprintf("provider=tencent 端口 %d: 撤销旧规则失败(index=%d): %v", targetPort, policyIndex, err))
				}
			}
		} else {
			config.Log("INFO", fmt.Sprintf("provider=tencent 端口 %d: 未找到现有规则，将添加新规则", targetPort))
		}

		config.Log("INFO", fmt.Sprintf("provider=tencent 端口 %d: 添加新规则 %s", targetPort, newCidr))
		createReq := tencentVPC.NewCreateSecurityGroupPoliciesRequest()
		createReq.SecurityGroupId = tencentCommon.StringPtr(securityGroupID)
		createReq.SecurityGroupPolicySet = &tencentVPC.SecurityGroupPolicySet{
			Ingress: []*tencentVPC.SecurityGroupPolicy{
				{
					Action:            tencentCommon.StringPtr("ACCEPT"),
					Protocol:          tencentCommon.StringPtr("TCP"),
					Port:              tencentCommon.StringPtr(strconv.Itoa(targetPort)),
					CidrBlock:         tencentCommon.StringPtr(newCidr),
					PolicyDescription: tencentCommon.StringPtr(fmt.Sprintf("白名单访问(端口%d) - Go脚本自动更新", targetPort)),
				},
			},
		}
		if _, err = client.CreateSecurityGroupPolicies(createReq); err != nil {
			config.Log("ERROR", fmt.Sprintf("provider=tencent 端口 %d: 授权失败: %v", targetPort, err))
		} else {
			config.Log("INFO", fmt.Sprintf("✓ provider=tencent 端口 %d: 已更新允许 %s", targetPort, newCidr))
		}
	}
}

func updateAliyunSecurityGroup(settings *models.Settings, currentIP string, ports []int) {
	region := strings.TrimSpace(settings.AliyunRegion)
	securityGroupID := strings.TrimSpace(settings.AliyunSecurityGroupID)

	openAPIConfig := &aliyunOpenAPI.Config{}
	openAPIConfig.SetAccessKeyId(strings.TrimSpace(settings.AliyunAccessKey))
	openAPIConfig.SetAccessKeySecret(strings.TrimSpace(settings.AliyunSecretKey))
	openAPIConfig.SetRegionId(region)

	client, err := aliyunECS.NewClient(openAPIConfig)
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("provider=aliyun: 创建阿里云会话失败: %v", err))
		return
	}

	newCidr := fmt.Sprintf("%s/32", currentIP)

	for _, targetPort := range ports {
		ingressRules, err := describeAliyunIngressRules(client, region, securityGroupID)
		if err != nil {
			config.Log("ERROR", fmt.Sprintf("provider=aliyun: 获取安全组规则失败: %v", err))
			return
		}

		matchedRules := findManagedAliyunIngressRules(ingressRules, targetPort)
		if isAliyunPortSynced(matchedRules, targetPort, newCidr) {
			config.Log("INFO", fmt.Sprintf("provider=aliyun 端口 %d: IP未变 (%s)，无需更新", targetPort, currentIP))
			continue
		}

		if len(matchedRules) > 0 {
			revokeAliyunIngressRules(client, region, securityGroupID, matchedRules, targetPort)
		} else {
			config.Log("INFO", fmt.Sprintf("provider=aliyun 端口 %d: 未找到现有规则，将添加新规则", targetPort))
		}

		config.Log("INFO", fmt.Sprintf("provider=aliyun 端口 %d: 添加新规则 %s", targetPort, newCidr))
		authorizeReq := &aliyunECS.AuthorizeSecurityGroupRequest{}
		authorizeReq.SetRegionId(region)
		authorizeReq.SetSecurityGroupId(securityGroupID)
		authorizeReq.SetPermissions([]*aliyunECS.AuthorizeSecurityGroupRequestPermissions{
			new(aliyunECS.AuthorizeSecurityGroupRequestPermissions).
				SetSourceCidrIp(newCidr).
				SetIpProtocol("TCP").
				SetPortRange(fmt.Sprintf("%d/%d", targetPort, targetPort)).
				SetPolicy("Accept").
				SetPriority("1").
				SetNicType("intranet").
				SetDescription(fmt.Sprintf("白名单访问(端口%d) - Go脚本自动更新", targetPort)),
		})

		if _, err = client.AuthorizeSecurityGroup(authorizeReq); err != nil {
			config.Log("ERROR", fmt.Sprintf("provider=aliyun 端口 %d: 授权失败: %v", targetPort, err))
		} else {
			config.Log("INFO", fmt.Sprintf("✓ provider=aliyun 端口 %d: 已更新允许 %s", targetPort, newCidr))
		}
	}
}

func describeAliyunIngressRules(client *aliyunECS.Client, region, securityGroupID string) ([]*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission, error) {
	allRules := make([]*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission, 0, 64)
	nextToken := ""

	for {
		req := &aliyunECS.DescribeSecurityGroupAttributeRequest{}
		req.SetRegionId(region)
		req.SetSecurityGroupId(securityGroupID)
		req.SetDirection("ingress")
		req.SetNicType("intranet")
		req.SetMaxResults(1000)
		if nextToken != "" {
			req.SetNextToken(nextToken)
		}

		resp, err := client.DescribeSecurityGroupAttribute(req)
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil {
			return allRules, nil
		}
		if resp.Body.Permissions != nil && len(resp.Body.Permissions.Permission) > 0 {
			allRules = append(allRules, resp.Body.Permissions.Permission...)
		}

		nextToken = strings.TrimSpace(aliyunStringValue(resp.Body.NextToken))
		if nextToken == "" {
			break
		}
	}

	return allRules, nil
}

func revokeAliyunIngressRules(client *aliyunECS.Client, region, securityGroupID string, rules []*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission, targetPort int) {
	ruleIDs := collectAliyunRuleIDs(rules)
	if len(ruleIDs) > 0 {
		config.Log("INFO", fmt.Sprintf("provider=aliyun 端口 %d: 撤销旧规则 %d 条", targetPort, len(ruleIDs)))
		revokeReq := &aliyunECS.RevokeSecurityGroupRequest{}
		revokeReq.SetRegionId(region)
		revokeReq.SetSecurityGroupId(securityGroupID)
		revokeReq.SetSecurityGroupRuleId(ruleIDs)
		if _, err := client.RevokeSecurityGroup(revokeReq); err != nil {
			config.Log("WARNING", fmt.Sprintf("provider=aliyun 端口 %d: 按规则ID撤销失败: %v", targetPort, err))
		}
	}

	legacyPermissions := buildAliyunRevokePermissionsWithoutID(rules)
	if len(legacyPermissions) > 0 {
		config.Log("INFO", fmt.Sprintf("provider=aliyun 端口 %d: 兼容模式撤销旧规则 %d 条", targetPort, len(legacyPermissions)))
		revokeReq := &aliyunECS.RevokeSecurityGroupRequest{}
		revokeReq.SetRegionId(region)
		revokeReq.SetSecurityGroupId(securityGroupID)
		revokeReq.SetPermissions(legacyPermissions)
		if _, err := client.RevokeSecurityGroup(revokeReq); err != nil {
			config.Log("WARNING", fmt.Sprintf("provider=aliyun 端口 %d: 兼容模式撤销失败: %v", targetPort, err))
		}
	}
}

func collectAliyunRuleIDs(rules []*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission) []*string {
	ruleIDs := make([]*string, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))

	for _, rule := range rules {
		if rule == nil || rule.SecurityGroupRuleId == nil {
			continue
		}
		ruleID := strings.TrimSpace(aliyunStringValue(rule.SecurityGroupRuleId))
		if ruleID == "" {
			continue
		}
		if _, ok := seen[ruleID]; ok {
			continue
		}
		seen[ruleID] = struct{}{}
		ruleIDs = append(ruleIDs, rule.SecurityGroupRuleId)
	}

	return ruleIDs
}

func buildAliyunRevokePermissionsWithoutID(rules []*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission) []*aliyunECS.RevokeSecurityGroupRequestPermissions {
	permissions := make([]*aliyunECS.RevokeSecurityGroupRequestPermissions, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))

	for _, rule := range rules {
		if rule == nil || rule.SecurityGroupRuleId != nil {
			continue
		}

		sourceCidr := strings.TrimSpace(aliyunStringValue(rule.SourceCidrIp))
		ipProtocol := strings.ToUpper(strings.TrimSpace(aliyunStringValue(rule.IpProtocol)))
		portRange := strings.TrimSpace(aliyunStringValue(rule.PortRange))
		if sourceCidr == "" || ipProtocol == "" || portRange == "" {
			continue
		}

		policy := strings.TrimSpace(aliyunStringValue(rule.Policy))
		if policy == "" {
			policy = "Accept"
		}
		priority := strings.TrimSpace(aliyunStringValue(rule.Priority))
		if priority == "" {
			priority = "1"
		}
		nicType := strings.TrimSpace(aliyunStringValue(rule.NicType))
		if nicType == "" {
			nicType = "intranet"
		}
		description := strings.TrimSpace(aliyunStringValue(rule.Description))

		uniqueKey := strings.Join([]string{sourceCidr, ipProtocol, portRange, policy, priority, nicType, description}, "|")
		if _, ok := seen[uniqueKey]; ok {
			continue
		}
		seen[uniqueKey] = struct{}{}

		permission := new(aliyunECS.RevokeSecurityGroupRequestPermissions).
			SetSourceCidrIp(sourceCidr).
			SetIpProtocol(ipProtocol).
			SetPortRange(portRange).
			SetPolicy(policy).
			SetPriority(priority).
			SetNicType(nicType)
		if description != "" {
			permission.SetDescription(description)
		}

		permissions = append(permissions, permission)
	}

	return permissions
}

func findManagedAliyunIngressRules(rules []*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission, targetPort int) []*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission {
	matched := make([]*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission, 0, 4)
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(aliyunStringValue(rule.Policy)), "accept") {
			continue
		}
		if strings.TrimSpace(aliyunStringValue(rule.SourceCidrIp)) == "" {
			continue
		}

		protocol := strings.ToUpper(strings.TrimSpace(aliyunStringValue(rule.IpProtocol)))
		if protocol != "TCP" && protocol != "ALL" {
			continue
		}
		if !aliyunRulePortContainsTarget(rule.PortRange, targetPort, protocol == "ALL") {
			continue
		}

		matched = append(matched, rule)
	}

	return matched
}

func isAliyunPortSynced(rules []*aliyunECS.DescribeSecurityGroupAttributeResponseBodyPermissionsPermission, targetPort int, targetCidr string) bool {
	if len(rules) != 1 {
		return false
	}

	rule := rules[0]
	if !strings.EqualFold(strings.TrimSpace(aliyunStringValue(rule.Policy)), "accept") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(aliyunStringValue(rule.IpProtocol)), "tcp") {
		return false
	}
	if !aliyunRulePortIsExact(rule.PortRange, targetPort) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(aliyunStringValue(rule.SourceCidrIp)), targetCidr) {
		return false
	}

	return true
}

func aliyunRulePortContainsTarget(portRangePtr *string, targetPort int, allowAll bool) bool {
	portRange := strings.TrimSpace(aliyunStringValue(portRangePtr))
	if portRange == "" {
		return allowAll
	}
	if portRange == "-1/-1" {
		return allowAll
	}

	fromPort, toPort, ok := parseAliyunPortRange(portRange)
	if !ok {
		return false
	}

	return targetPort >= fromPort && targetPort <= toPort
}

func aliyunRulePortIsExact(portRangePtr *string, targetPort int) bool {
	portRange := strings.TrimSpace(aliyunStringValue(portRangePtr))
	if portRange == "" || portRange == "-1/-1" {
		return false
	}

	fromPort, toPort, ok := parseAliyunPortRange(portRange)
	if !ok {
		return false
	}

	return fromPort == targetPort && toPort == targetPort
}

func parseAliyunPortRange(portRange string) (int, int, bool) {
	portRange = strings.TrimSpace(portRange)
	if portRange == "" {
		return 0, 0, false
	}
	if strings.Contains(portRange, "/") {
		parts := strings.SplitN(portRange, "/", 2)
		if len(parts) != 2 {
			return 0, 0, false
		}

		fromPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || fromPort <= 0 || fromPort > 65535 {
			return 0, 0, false
		}
		toPort, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || toPort <= 0 || toPort > 65535 {
			return 0, 0, false
		}
		if fromPort > toPort {
			return 0, 0, false
		}
		return fromPort, toPort, true
	}

	value, err := strconv.Atoi(portRange)
	if err != nil || value <= 0 || value > 65535 {
		return 0, 0, false
	}
	return value, value, true
}

func aliyunStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func describeTencentIngressPolicies(client *tencentVPC.Client, securityGroupID string) ([]*tencentVPC.SecurityGroupPolicy, error) {
	req := tencentVPC.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = tencentCommon.StringPtr(securityGroupID)

	resp, err := client.DescribeSecurityGroupPolicies(req)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Response == nil || resp.Response.SecurityGroupPolicySet == nil {
		return nil, nil
	}
	return resp.Response.SecurityGroupPolicySet.Ingress, nil
}

func findManagedTencentIngressRules(rules []*tencentVPC.SecurityGroupPolicy, targetPort int) []*tencentVPC.SecurityGroupPolicy {
	matched := make([]*tencentVPC.SecurityGroupPolicy, 0, 4)
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(tencentStringValue(rule.Action)), "ACCEPT") {
			continue
		}
		if strings.TrimSpace(tencentStringValue(rule.CidrBlock)) == "" {
			continue
		}

		protocol := strings.ToUpper(strings.TrimSpace(tencentStringValue(rule.Protocol)))
		if protocol != "TCP" && protocol != "ALL" {
			continue
		}
		if !tencentRulePortContainsTarget(rule.Port, targetPort, protocol == "ALL") {
			continue
		}

		matched = append(matched, rule)
	}

	return matched
}

func isTencentPortSynced(rules []*tencentVPC.SecurityGroupPolicy, targetPort int, targetCidr string) bool {
	if len(rules) != 1 {
		return false
	}

	rule := rules[0]
	if !strings.EqualFold(strings.TrimSpace(tencentStringValue(rule.Action)), "ACCEPT") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(tencentStringValue(rule.Protocol)), "TCP") {
		return false
	}
	if !tencentRulePortIsExact(rule.Port, targetPort) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(tencentStringValue(rule.CidrBlock)), targetCidr) {
		return false
	}

	return true
}

func collectTencentPolicyIndexes(rules []*tencentVPC.SecurityGroupPolicy) []int64 {
	indexes := make([]int64, 0, len(rules))
	seen := make(map[int64]struct{}, len(rules))

	for _, rule := range rules {
		if rule == nil || rule.PolicyIndex == nil {
			continue
		}
		index := *rule.PolicyIndex
		if _, ok := seen[index]; ok {
			continue
		}
		seen[index] = struct{}{}
		indexes = append(indexes, index)
	}

	return indexes
}

func tencentRulePortContainsTarget(portPtr *string, targetPort int, allowAll bool) bool {
	port := strings.TrimSpace(tencentStringValue(portPtr))
	if port == "" {
		return allowAll
	}
	if strings.EqualFold(port, "all") {
		return true
	}

	fromPort, toPort, ok := parseTencentPortRange(port)
	if !ok {
		return false
	}
	return targetPort >= fromPort && targetPort <= toPort
}

func tencentRulePortIsExact(portPtr *string, targetPort int) bool {
	port := strings.TrimSpace(tencentStringValue(portPtr))
	if port == "" || strings.EqualFold(port, "all") {
		return false
	}

	fromPort, toPort, ok := parseTencentPortRange(port)
	if !ok {
		return false
	}
	return fromPort == targetPort && toPort == targetPort
}

func parseTencentPortRange(port string) (int, int, bool) {
	port = strings.TrimSpace(port)
	if port == "" {
		return 0, 0, false
	}

	if strings.Contains(port, "-") {
		parts := strings.SplitN(port, "-", 2)
		if len(parts) != 2 {
			return 0, 0, false
		}
		fromPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || fromPort <= 0 || fromPort > 65535 {
			return 0, 0, false
		}
		toPort, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || toPort <= 0 || toPort > 65535 {
			return 0, 0, false
		}
		if fromPort > toPort {
			return 0, 0, false
		}
		return fromPort, toPort, true
	}

	value, err := strconv.Atoi(port)
	if err != nil || value <= 0 || value > 65535 {
		return 0, 0, false
	}
	return value, value, true
}

func tencentStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func findManagedAWSEC2Rules(rules []*awsec2.SecurityGroupRule, targetPort int) []*awsec2.SecurityGroupRule {
	matched := make([]*awsec2.SecurityGroupRule, 0, 4)
	for _, rule := range rules {
		if rule == nil || awsSDK.BoolValue(rule.IsEgress) {
			continue
		}

		protocol := strings.ToLower(awsSDK.StringValue(rule.IpProtocol))
		if protocol != "tcp" && protocol != "-1" {
			continue
		}

		fromPort := int(awsSDK.Int64Value(rule.FromPort))
		toPort := int(awsSDK.Int64Value(rule.ToPort))
		if protocol == "-1" {
			// all protocol rule should be considered matched
			matched = append(matched, rule)
			continue
		}

		if targetPort < fromPort || targetPort > toPort {
			continue
		}

		matched = append(matched, rule)
	}
	return matched
}

func isAWSEC2PortSynced(rules []*awsec2.SecurityGroupRule, targetPort int, targetCidr string) bool {
	if len(rules) != 1 {
		return false
	}

	rule := rules[0]
	if strings.ToLower(awsSDK.StringValue(rule.IpProtocol)) != "tcp" {
		return false
	}
	if int(awsSDK.Int64Value(rule.FromPort)) != targetPort || int(awsSDK.Int64Value(rule.ToPort)) != targetPort {
		return false
	}
	if !strings.EqualFold(awsSDK.StringValue(rule.CidrIpv4), targetCidr) {
		return false
	}
	if awsSDK.BoolValue(rule.IsEgress) {
		return false
	}
	return true
}

func normalizeAWSRegion(region string) (string, bool) {
	region = strings.ToLower(strings.TrimSpace(region))
	parts := strings.Split(region, "-")
	if len(parts) < 3 {
		return region, false
	}

	last := parts[len(parts)-1]
	if len(last) != 2 {
		return region, false
	}
	zoneSuffix := last[1]
	if zoneSuffix < 'a' || zoneSuffix > 'z' {
		return region, false
	}
	if last[0] < '0' || last[0] > '9' {
		return region, false
	}

	parts[len(parts)-1] = last[:1]
	return strings.Join(parts, "-"), true
}

func findManagedLightsailStates(portStates []*awslightsail.InstancePortState, targetPort int) []*awslightsail.InstancePortState {
	matched := make([]*awslightsail.InstancePortState, 0, 2)
	for _, state := range portStates {
		if state == nil {
			continue
		}

		fromPort := int(awsSDK.Int64Value(state.FromPort))
		toPort := int(awsSDK.Int64Value(state.ToPort))
		if targetPort < fromPort || targetPort > toPort {
			continue
		}

		protocol := strings.ToLower(awsSDK.StringValue(state.Protocol))
		if protocol == awslightsail.NetworkProtocolTcp || protocol == awslightsail.NetworkProtocolAll {
			matched = append(matched, state)
		}
	}
	return matched
}

func isLightsailPortSynced(states []*awslightsail.InstancePortState, targetPort int, targetCidr string) bool {
	if len(states) != 1 {
		return false
	}

	state := states[0]
	if int(awsSDK.Int64Value(state.FromPort)) != targetPort || int(awsSDK.Int64Value(state.ToPort)) != targetPort {
		return false
	}
	if strings.ToLower(awsSDK.StringValue(state.Protocol)) != awslightsail.NetworkProtocolTcp {
		return false
	}
	if strings.ToLower(awsSDK.StringValue(state.State)) != "open" {
		return false
	}
	if len(state.Cidrs) != 1 || len(state.CidrListAliases) > 0 || len(state.Ipv6Cidrs) > 0 {
		return false
	}

	return strings.EqualFold(awsSDK.StringValue(state.Cidrs[0]), targetCidr)
}
