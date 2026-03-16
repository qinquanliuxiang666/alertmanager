# alert

## 告警流程

```mermaid
flowchart TD
    %% 定义节点样式
    classDef endpoint fill:#f9f,stroke:#333,stroke-width:2px;
    classDef controller fill:#bbf,stroke:#333,stroke-width:1px;
    classDef filter fill:#ffcccb,stroke:#333,stroke-width:1px;
    classDef split fill:#e2e3e5,stroke:#333,stroke-width:2px;
    classDef firing fill:#f8d7da,stroke:#333,stroke-width:1px;
    classDef resolved fill:#d4edda,stroke:#333,stroke-width:1px;
    classDef db fill:#fcf8e3,stroke:#333,stroke-width:1px;
    classDef external fill:#d1ecf1,stroke:#333,stroke-width:1px;
    classDef async fill:#ffeeba,stroke:#333,stroke-width:1px,stroke-dasharray: 5 5;

    %% 1. 接收与解析层
    A([Alertmanager Webhook]) -->|POST /api/v1/alerts?channel_name=feishu| B["接收端点"]:::endpoint
    B --> C["Controller 层<br/>1. 解析 URL 参数中的 ChannelID<br/>2. 反序列化告警 Payload"]:::controller

    %% 2. 预处理与分流
    C --> D{"本地全局静默检查<br/>(可选: 平台级屏蔽)"}:::filter
    D -- "命中屏蔽" --> E["丢弃告警 / 审计记录"]:::filter
    D -- "有效告警" --> F{"按状态拆分"}:::split

    F -->|Firing 数组| F1["获取 Channel 配置<br/>(根据 URL 传参 ID 直接查询 DB)"]:::firing
    F -->|Resolved 数组| R1["处理恢复逻辑"]:::resolved

    %% ================= 触发流水线 (Firing Pipeline) =================
    F1 --> F5{"告警聚合模块 (Grouping)<br/>按 GroupKey 入桶等待<br/>(防止并发抖动)"}:::firing

    F5 -- "窗口闭合 (Batch)" --> F6["加载 Channel 关联模板<br/>执行模板渲染"]:::firing

    %% 这里体现你之前的需求：如果有旧卡片则更新，没有则发送
    F6 --> F_Check{"DB 查询:<br/>是否存在活跃 GroupKey?"}:::db

    F_Check -- "不存在" --> F7["调用 API 发送新消息"]:::external
    F_Check -- "存在旧卡片" --> F_Update["调用 API 更新原消息<br/>+ 回复 Thread 提醒"]:::external

    F7 --> F8["入库: AlertHistory & AlertSendRecord"]:::db
    F_Update --> F8

    %% ================= 恢复流水线 (Resolved Pipeline) =================
    R1 --> R2["根据指纹 (Fingerprint) 查询 DB<br/>定位原 MessageID"]:::db

    R2 --> R3{"按 MessageID 聚合<br/>(快照重绘)"}:::resolved

    R3 --> R4["模板渲染 (绿卡/删除线)"]:::resolved

    R4 --> R5["调用 API 更新(Patch)原消息<br/>+ 回复 Thread 提醒"]:::external

    R5 --> R6["更新 DB: AlertHistory 状态及 EndsAt"]:::db

    %% ================= 异步告警升级流程 (Escalation) =================
    %% 升级逻辑依然保留在平台内，因为这属于“发送后”的状态追踪
    T([定时任务 Cron]) -.-> U{"扫描 DB 中 Firing 状态<br/>且超时的告警记录"}:::async
    U -- "满足升级条件" --> V["获取 Channel 关联的升级配置"]:::async
    V --> W["调用第三方 API (如语音电话)"]:::external
    W --> X["记录升级日志"]:::db
```

## DispatchAlert

```go
func DispatchAlert(db *gorm.DB, incomingAlert AlertHistory) error {
	// 1. 获取所有启用的路由规则（按优先级排序）
	var rules[]model.AlertRouteRule
	db.Preload("Channels"). // 重点：把关联的发送渠道一起查出来
		Where("status = ?", 1).
		Order("priority DESC").
		Find(&rules)

	// 2. 遍历规则，看这个告警符合哪个规则
	for _, rule := range rules {
		// 解析规则的匹配条件
		var conditions[]configs.MatchCondition
		json.Unmarshal(rule.MatchConditions, &conditions)

		// 检查告警标签是否满足所有条件
		if isMatch(incomingAlert.Labels, conditions) {

			// 3. 一旦匹配成功，获取绑定的发送渠道并发送
			for _, channel := range rule.Channels {
				// 调用之前讨论的通知器工厂发送告警
				notifier := notifier.BuildNotifier(channel.Type, channel.Config)
				notifier.Send(incomingAlert.Alertname, "告警详情...")
			}

			// 匹配到一条规则并发送后，通常可以选择停止匹配后续规则（防止重复发送）
			break
		}
	}
	return nil
}

// isMatch 函数实现标签匹配逻辑
func isMatch(alertLabelsJSON datatypes.JSON, conditions []configs.MatchCondition) bool {
    // 逻辑：将 alertLabels 解析为 map[string]string
    // 遍历 conditions，检查 map 中对应 key 的 value 是否满足 Operator 和 Value 的要求
    // 全部满足返回 true，否则返回 false
    return true
}
```

## 飞书卡片

```go
package configs

// FeishuCardMsg 飞书卡片消息的外层结构
type FeishuCardMsg struct {
	MsgType string     `json:"msg_type"` // 固定值: "interactive"
	Card    FeishuCard `json:"card"`
}

type FeishuCard struct {
	Config   FeishuCardConfig    `json:"config"`
	Header   FeishuCardHeader    `json:"header"`
	Elements[]FeishuCardElement `json:"elements"`
}

type FeishuCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"` // 宽屏模式
}

type FeishuCardHeader struct {
	Template string `json:"template"` // 颜色：red, green, blue 等
	Title    struct {
		Content string `json:"content"`
		Tag     string `json:"tag"` // "plain_text"
	} `json:"title"`
}

type FeishuCardElement struct {
	Tag     string `json:"tag"`     // "markdown", "div" 等
	Content string `json:"content"` // Markdown 文本内容
}

package notifier

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"
	"time"

	"your_project/model"
	"your_project/configs"
)

// 定义模板支持的自定义函数（比如把 firing 变成大写）
var tplFuncs = template.FuncMap{
	"ToUpper": strings.ToUpper,
}

// RenderAndSendFeishu 渲染模板并组装飞书卡片发送
func RenderAndSendFeishu(alert model.AlertHistory, tpl model.AlertTemplate, webhookURL string) error {
	// 1. 编译并渲染 Title
	titleT, err := template.New("title").Funcs(tplFuncs).Parse(tpl.TitleTpl)
	if err != nil {
		return err
	}
	var titleBuf bytes.Buffer
	titleT.Execute(&titleBuf, alert)

	// 2. 编译并渲染 Body (Markdown内容)
	bodyT, err := template.New("body").Funcs(tplFuncs).Parse(tpl.BodyTpl)
	if err != nil {
		return err
	}
	var bodyBuf bytes.Buffer
	bodyT.Execute(&bodyBuf, alert)

	// 3. 决定卡片的颜色 (状态为 firing 用红色，resolved 用绿色)
	headerColor := "red"
	if alert.Status == "resolved" {
		headerColor = "green"
	}

	// 4. 安全地组装飞书结构体 (避免了 JSON 注入)
	feishuMsg := configs.FeishuCardMsg{
		MsgType: "interactive",
		Card: configs.FeishuCard{
			Config: configs.FeishuCardConfig{WideScreenMode: true},
			Header: configs.FeishuCardHeader{
				Template: headerColor, // 动态颜色
				Title: struct {
					Content string `json:"content"`
					Tag     string `json:"tag"`
				}{
					Content: titleBuf.String(), // 注入渲染好的标题
					Tag:     "plain_text",
				},
			},
			Elements:
```
