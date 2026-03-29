package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	"github.com/qinquanliuxiang666/alertmanager/base/conf"
	"github.com/qinquanliuxiang666/alertmanager/base/constant"
	"github.com/qinquanliuxiang666/alertmanager/base/helper"
	"github.com/qinquanliuxiang666/alertmanager/base/log"
	"github.com/qinquanliuxiang666/alertmanager/base/types"
	"github.com/qinquanliuxiang666/alertmanager/model"
	"github.com/qinquanliuxiang666/alertmanager/pkg/alert"
)

var feishuStruct = &FeiShu{
	clients: make(map[string]*Client),
}

func printFeishuCli() string {
	clientNames := make([]string, 0, len(feishuStruct.clients))
	for name := range feishuStruct.clients {
		clientNames = append(clientNames, name)
	}
	return strings.Join(clientNames, ",")
}

type Feishuer interface {
	Init(alertChannelName, appid, appSecret string)
	GetCli(alertChannelName string) (*lark.Client, error)
	UpdateCli(alertChannelName, appid, appSecret string)
	CloseCli(alertChannelName string)
	FeishuCarder
}

type FeishuCarder interface {
	SendCard(ctx context.Context, alertChannel *model.AlertChannel, req *types.AlertReceiveReq) error
}

type FeiShu struct {
	lock        sync.Mutex
	clients     map[string]*Client
	AlertUtiler alert.AlertUtiler
}

type Client struct {
	cli      *lark.Client
	wsCli    *larkws.Client
	cancelFn context.CancelFunc
}

func NewFeiShu(alertUtiler alert.AlertUtiler) Feishuer {
	feishuStruct.AlertUtiler = alertUtiler
	return feishuStruct
}

func (receiver *FeiShu) Init(alertChannelName, appid, appSecret string) {
	receiver.lock.Lock()
	defer receiver.lock.Unlock()
	if _, ok := feishuStruct.clients[alertChannelName]; ok {
		return
	}

	cli, wsCli, cancelFn := newFeishuClient(alertChannelName, appid, appSecret)
	receiver.clients[alertChannelName] = &Client{
		cli:      cli,
		wsCli:    wsCli,
		cancelFn: cancelFn,
	}
	clientNames := printFeishuCli()
	zap.S().Infof("初始化新的飞书客户端 %s, 当前已缓存的客户端 %s", alertChannelName, clientNames)
}

func (receiver *FeiShu) GetCli(alertChannelName string) (*lark.Client, error) {
	receiver.lock.Lock()
	defer receiver.lock.Unlock()

	c, ok := receiver.clients[alertChannelName]
	if !ok {
		return nil, fmt.Errorf("client %s not initialized", alertChannelName)
	}
	return c.cli, nil
}

func newFeishuClient(alertChannelName, appid, appSecret string) (*lark.Client, *larkws.Client, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
			feiShuCardTrigger := new(helper.FeiShuCardTrigger)
			if err := json.Unmarshal(event.Body, feiShuCardTrigger); err != nil {
				return nil, err
			}
			fmt.Println("☀️------------------------------------☀️")
			v, ok := feiShuCardTrigger.Event.Action.Value[feiShuCardTrigger.Event.Action.Option]
			if ok {
				fmt.Printf("v: %s", v)
			}
			fmt.Println("🌙------------------------------------🌙")

			return nil, nil
			// return &callback.CardActionTriggerResponse{
			// 	Toast: &callback.Toast{
			// 		Type:    "info",
			// 		Content: "静默成功",
			// 	},
			// 	Card: &callback.Card{
			// 		Type: "template",
			// 		Data: map[string]any{
			// 			"template_id": "AAqK947a7l70i",
			// 			"template_variable": map[string]any{
			// 				"disableSelect": true,
			// 			},
			// 		},
			// 	},
			// }, nil
		}).
		// 监听「拉取链接预览数据 url.preview.get」
		OnP2CardURLPreviewGet(func(ctx context.Context, event *callback.URLPreviewGetEvent) (*callback.URLPreviewGetResponse, error) {
			// fmt.Printf("[ OnP2URLPreviewAction access ], data: %s\n", larkcore.Prettify(event))
			evebtByte, err := json.Marshal(event)
			if err != nil {
				panic(err)
			}

			fmt.Println("☀️------------------------------------☀️")
			fmt.Println(string(evebtByte))
			fmt.Println("🌙------------------------------------🌙")
			return nil, nil
		})
	// 创建Client

	var larkLogLevel larkcore.LogLevel
	if conf.GetLogLevel() == "debug" {
		larkLogLevel = larkcore.LogLevelDebug
	} else {
		larkLogLevel = larkcore.LogLevelInfo
	}

	wsCli := larkws.NewClient(appid, appSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkLogLevel),
	)
	zap.S().Infof("创建新的飞书客户端 %s 长连接", alertChannelName)

	go func() {
		err := wsCli.Start(ctx)
		if err != nil {
			if err == context.Canceled {
				zap.L().Info("lark WS Connection closed by cancelFn", zap.String("app_id", appid))
				return
			}
			zap.L().Error("lark WS Start Error", zap.Error(err))
		}
	}()

	cli := lark.NewClient(appid, appSecret,
		lark.WithLogLevel(larkcore.LogLevelDebug),
		lark.WithReqTimeout(10*time.Second),
	)

	clientNames := printFeishuCli()
	zap.S().Infof("创建新的飞书客户端 %s, 当前已缓存的客户端 %s", alertChannelName, clientNames)
	return cli, wsCli, cancel
}

// UpdateCli 如果 appid 和 appSecret 修改需要重新初始化客户端
func (receiver *FeiShu) UpdateCli(alertChannelName, appid, appSecret string) {
	receiver.lock.Lock()
	defer receiver.lock.Unlock()

	// 1. 如果旧客户端存在，先执行销毁逻辑
	if oldClient, ok := receiver.clients[alertChannelName]; ok {
		zap.S().Info("正在更新通道 [%s] 的客户端，关闭旧连接...", alertChannelName)
		if oldClient.cancelFn != nil {
			// 触发 context 取消，停止旧的 wsCli.Start
			oldClient.cancelFn()
			zap.S().Infof("停止旧的飞书客户端 %s 连接, ", alertChannelName)
		}
	}

	time.Sleep(5 * time.Second)
	// 2. 创建新的客户端
	cli, wsCli, cancel := newFeishuClient(alertChannelName, appid, appSecret)

	// 3. 覆盖存储
	receiver.clients[alertChannelName] = &Client{
		cli:      cli,
		wsCli:    wsCli,
		cancelFn: cancel,
	}
	clientNames := printFeishuCli()
	zap.S().Infof("已更新飞书客户端 %s, 当前已缓存的客户端 %s", alertChannelName, clientNames)
}

// UpdateCli 如果 appid 和 appSecret 修改需要重新初始化客户端
func (receiver *FeiShu) CloseCli(alertChannelName string) {
	receiver.lock.Lock()
	defer receiver.lock.Unlock()

	// 1. 如果旧客户端存在，先执行销毁逻辑
	if oldClient, ok := receiver.clients[alertChannelName]; ok {
		zap.S().Infof("正在更新通道 [%s] 的客户端，关闭旧连接...", alertChannelName)
		if oldClient.cancelFn != nil {
			oldClient.cancelFn()
			time.Sleep(5 * time.Second)
		}
	}
	delete(receiver.clients, alertChannelName)
	clientNames := printFeishuCli()
	zap.S().Infof("从本地缓存中删除 [%s] 的客户端成功, 当前已缓存的客户端 %s", alertChannelName, clientNames)
}

func (receiver *FeiShu) SendCard(ctx context.Context, alertChannel *model.AlertChannel, req *types.AlertReceiveReq) error {
	if alertChannel.AlertTemplate == nil {
		return fmt.Errorf("%s alertChannel 未绑定模板, 发送告警失败", alertChannel.Name)
	}
	feishuAppConf, err := alertChannel.GetFeishuAppConfig()
	if err != nil {
		return fmt.Errorf("获取飞书配置失败: %w", err)
	}

	larkCli, err := receiver.GetCli(req.ChannelName)
	if err != nil {
		return err
	}

	if *alertChannel.AggregationStatus == model.AggregationEnabled {
		sendAlerts, err := receiver.handleAggregation(ctx, larkCli, feishuAppConf, alertChannel, req)
		if err != nil {
			return err
		}
		receiver.AlertUtiler.SaveAggregationAlert(ctx, alertChannel, sendAlerts)
	} else {
		normalSendResult, err := receiver.handleNormal(ctx, larkCli, feishuAppConf, alertChannel, req)
		if err != nil {
			return err
		}
		receiver.AlertUtiler.SaveNormalAlerts(ctx, alertChannel, normalSendResult)
	}
	return nil
}

func (receiver *FeiShu) handleAggregation(ctx context.Context, larkCli *lark.Client, conf *model.FeishuAppConfig, alertChannel *model.AlertChannel, req *types.AlertReceiveReq) (*types.HandleAggregationSendResult, error) {
	firingAlerts, resolvedAlerts := receiver.AlertUtiler.AlertGroup(ctx, req.Alerts)
	var firingErr, resolvedErr error
	// 处理告警中 (Firing)
	if len(firingAlerts) > 0 {
		newReq := req.DeepCopy()
		newReq.Alerts = firingAlerts
		if firingErr = receiver.renderAndSend(ctx, larkCli, conf, newReq, alertChannel.AlertTemplate.AggregationTemplate, "red"); firingErr != nil {
			log.WithRequestID(ctx).Error("聚合发送告警卡片失败", zap.Error(firingErr))
		}
	}

	// 处理已恢复 (Resolved)
	if len(resolvedAlerts) > 0 {
		newReq := req.DeepCopy()
		newReq.Alerts = resolvedAlerts
		if resolvedErr = receiver.renderAndSend(ctx, larkCli, conf, newReq, alertChannel.AlertTemplate.AggregationTemplate, "green"); resolvedErr != nil {
			log.WithRequestID(ctx).Error("聚合发送恢复卡片失败", zap.Error(resolvedErr))
		}
	}

	return &types.HandleAggregationSendResult{
		FiringErr:      firingErr,
		ResolvedErr:    resolvedErr,
		FiringAlerts:   firingAlerts,
		ResolvedAlerts: resolvedAlerts,
	}, errors.Join(firingErr, resolvedErr)
}

func (receiver *FeiShu) handleNormal(ctx context.Context, larkCli *lark.Client, conf *model.FeishuAppConfig, alertChannel *model.AlertChannel, req *types.AlertReceiveReq) ([]*types.NormalSendResult, error) {
	var errs []error
	var results []*types.NormalSendResult
	for _, v := range req.Alerts {
		color := "red"
		if v.Status == constant.AlertStatusResolved {
			color = "green"
		}

		err := receiver.renderAndSend(ctx, larkCli, conf, v, alertChannel.AlertTemplate.Template, color)

		// 记录结果，用于后续落库
		results = append(results, &types.NormalSendResult{
			Alert:   v,
			SendErr: err,
		})

		if err != nil {
			log.WithRequestID(ctx).Error("发送单条飞书卡片失败", zap.Error(err))
			if len(errs) < 4 {
				errs = append(errs, err)
			}
			continue
		}
	}
	return results, errors.Join(errs...)
}

func (receiver *FeiShu) renderAndSend(ctx context.Context, larkCli *lark.Client, conf *model.FeishuAppConfig, data interface{}, tpl string, color string) error {
	// 1. 渲染模板
	content, err := RenderingAlertContent().Build(data, tpl)
	if err != nil {
		return err
	}

	// 2. 设置标题颜色 (如果模板里没写死的话)
	if content.Data.TemplateVariable == nil {
		content.Data.TemplateVariable = make(map[string]any)
	}
	content.Data.TemplateVariable["titleColor"] = color

	// 3. 序列化
	byData, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("marshal失败: %w", err)
	}

	// 4. 发送
	return SendCard(larkCli).Build(ctx, conf.ReceiveIdType, conf.ReceiveId, string(byData))
}

type FeishuCard struct {
	cli *lark.Client
}

func SendCard(cli *lark.Client) *FeishuCard {
	return &FeishuCard{
		cli: cli,
	}
}

func (receiver *FeishuCard) Build(ctx context.Context, receiveIdType, receiveId, content string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIdType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveId).
			MsgType(`interactive`).
			Content(content).
			Build()).
		Build()

	// 发起请求
	resp, err := receiver.cli.Im.V1.Message.Create(ctx, req)

	// 处理错误
	if err != nil {
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		log.WithRequestID(ctx).Error("发起请求发送飞书卡片时服务发生错误", zap.String("logId", resp.RequestId()), zap.Error(err), zap.String("错误响应", larkcore.Prettify(resp.CodeError)))
		return err
	}

	// // 业务处理
	// fmt.Println(larkcore.Prettify(resp))
	return nil
}

// FeiShuContent 飞书卡片模版request
type FeiShuContent struct {
	Type string                `json:"type"`
	Data FeishuCardDataContent `json:"data"`
}

type FeishuCardDataContent struct {
	TemplateId          string         `json:"template_id"  yaml:"template_id"`
	TemplateNersionName string         `json:"template_version_name" yaml:"template_version_name"`
	TemplateVariable    map[string]any `json:"template_variable" yaml:"template_variable"`
}

func RenderingAlertContent() *FeishuCardDataContent {
	return &FeishuCardDataContent{}
}

var funcMap = template.FuncMap{
	"timeFormat": func(t time.Time) string {
		var cstZone = time.FixedZone("CST", 8*3600)
		return t.In(cstZone).Format("2006-01-02 15:04:05")
	},
	"add": func(a, b int) int {
		return a + b
	},
	"getEndTime": func(endTime *time.Time, msg string) string {
		if endTime == nil || endTime.IsZero() {
			return msg
		}
		var cstZone = time.FixedZone("CST", 8*3600)
		return endTime.In(cstZone).Format("2006-01-02 15:04:05")
	},
}

func (receiver *FeishuCardDataContent) Build(alert any, alertTpl string) (*FeiShuContent, error) {
	tmpl, err := template.New("alert").Funcs(funcMap).Parse(alertTpl)
	if err != nil {
		return nil, fmt.Errorf("构建告警模版失败, %s", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, alert); err != nil {
		return nil, fmt.Errorf("渲染告警模版失败, %s", err)
	}

	if err := yaml.Unmarshal([]byte(buf.Bytes()), &receiver); err != nil {
		return nil, fmt.Errorf("序列化 FeishuCardDataContent 失败, %s", err)
	}

	return &FeiShuContent{
		Type: "template",
		Data: *receiver,
	}, nil
}
