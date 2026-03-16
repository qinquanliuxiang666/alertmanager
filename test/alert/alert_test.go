package alert_test

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// AlertmanagerPayload 对应你提供的 JSON 结构
type AlertmanagerPayload struct {
	Alerts            []Alert           `json:"alerts"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	CommonLabels      map[string]string `json:"commonLabels"`
	ExternalURL       string            `json:"externalURL"`
	GroupKey          string            `json:"groupKey"`
	GroupLabels       map[string]string `json:"groupLabels"`
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Version           string            `json:"version"`
}

type Alert struct {
	Annotations  map[string]string `json:"annotations"`
	EndsAt       time.Time         `json:"endsAt"`
	Fingerprint  string            `json:"fingerprint"`
	GeneratorURL string            `json:"generatorURL"`
	Labels       map[string]string `json:"labels"`
	StartsAt     time.Time         `json:"startsAt"`
	Status       string            `json:"status"`
}

// 模拟预设的数据池
var (
	alertNames = []string{"NodeDiskUsageHigh", "CPUThrottlingHigh", "MemoryLeakDetected", "ServiceDown", "KubePodCrashLooping"}
	severities = []string{"critical", "warning", "info"}
	teams      = []string{"infrastructure", "backend", "devops", "dba"}
)

// GenerateRandomAlerts 生成模拟数据
// totalAlerts: 总告警条数
// numGroups: 分成多少个组发送（返回一个切片，每个元素代表一个分组的 Payload）
func GenerateRandomAlerts(totalAlerts int, numGroups int) []AlertmanagerPayload {
	rand.Seed(time.Now().UnixNano())

	if numGroups <= 0 {
		numGroups = 1
	}
	alertsPerGroup := totalAlerts / numGroups

	var payloads []AlertmanagerPayload
	usedFingerprints := make(map[string]bool)

	for i := 0; i < numGroups; i++ {
		// 确定当前组的告警量
		count := alertsPerGroup
		if i == numGroups-1 { // 最后一组补齐余数
			count = totalAlerts - (alertsPerGroup * (numGroups - 1))
		}

		alertName := alertNames[rand.Intn(len(alertNames))]
		groupLabels := map[string]string{
			"alertname": alertName,
			"cluster":   "prod-aliyun-01",
		}

		payload := AlertmanagerPayload{
			Status:            "firing",
			Receiver:          "feishu-receiver",
			ExternalURL:       "http://alertmanager.example.com",
			Version:           "4",
			GroupLabels:       groupLabels,
			GroupKey:          fmt.Sprintf("{}/{alertname=%q, cluster=\"prod-aliyun-01\"}", alertName),
			CommonLabels:      groupLabels,
			CommonAnnotations: make(map[string]string),
			Alerts:            []Alert{},
		}

		for j := 0; j < count; j++ {
			instance := fmt.Sprintf("10.0.0.%d:9100", rand.Intn(254))
			startsAt := time.Now().Add(time.Duration(-rand.Intn(10000)) * time.Second)

			// 生成唯一的 Fingerprint: 基于实例名和时间戳生成 MD5
			hasher := md5.New()
			hasher.Write([]byte(fmt.Sprintf("%s-%d-%d", instance, startsAt.Unix(), rand.Int63())))
			fp := hex.EncodeToString(hasher.Sum(nil))[:16]

			// 确保唯一性（简单防重）
			for usedFingerprints[fp+startsAt.String()] {
				fp = fp[1:] + "1"
			}
			usedFingerprints[fp+startsAt.String()] = true

			alert := Alert{
				Status:       "firing",
				Fingerprint:  fp,
				StartsAt:     startsAt,
				EndsAt:       time.Time{}, // 0001-01-01
				GeneratorURL: fmt.Sprintf("http://vmalert:8080/vmalert/alert?id=%s", fp),
				Labels: map[string]string{
					"alertname": alertName,
					"instance":  instance,
					"severity":  severities[rand.Intn(len(severities))],
					"team":      teams[rand.Intn(len(teams))],
					"job":       "node-exporter",
					"device":    "/dev/sda1",
				},
				Annotations: map[string]string{
					"summary":     fmt.Sprintf("告警触发: %s 在 %s", alertName, instance),
					"description": fmt.Sprintf("检测到当前值 %d%% 超过阈值", rand.Intn(50)+50),
				},
			}
			payload.Alerts = append(payload.Alerts, alert)
		}
		payloads = append(payloads, payload)
	}

	return payloads
}

func TestAlert(t *testing.T) {
	// 配置参数
	totalAlerts := 500
	numGroups := 2
	outputDir := "alerts_output"

	// 1. 创建输出目录
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		return
	}

	// 2. 生成数据
	payloads := GenerateRandomAlerts(totalAlerts, numGroups)

	// 3. 循环写入文件
	for i, payload := range payloads {
		// 生成文件名: alert_group_1_2023...json
		fileName := fmt.Sprintf("group_%d_%s.json", i+1, payload.GroupLabels["alertname"])
		filePath := filepath.Join(outputDir, fileName)

		// 格式化 JSON
		fileData, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Printf("JSON 转换失败: %v\n", err)
			continue
		}

		// 写入磁盘
		err = os.WriteFile(filePath, fileData, 0644)
		if err != nil {
			fmt.Printf("文件 %s 写入失败: %v\n", filePath, err)
		} else {
			fmt.Printf("成功写入文件: %s (包含 %d 条告警)\n", filePath, len(payload.Alerts))
		}
	}

	fmt.Printf("\n所有告警已生成到目录: %s/\n", outputDir)

}
