package types

import (
	"time"
)

// AlertReceiveReq 是 Alertmanager 发送的 Webhook 顶层 JSON 结构
type AlertReceiveReq struct {
	ChannelName       string            `form:"channelName" binding:"required"`
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"` // "firing" or "resolved"
	Alerts            []*Alert          `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   uint64            `json:"truncatedAlerts"`
}

// Alert 代表单条告警的详情
type Alert struct {
	Status       string            `json:"status"` // "firing" or "resolved"
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       *time.Time        `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

func (receiver *AlertReceiveReq) DeepCopy() *AlertReceiveReq {
	return &AlertReceiveReq{
		ChannelName:       receiver.ChannelName,
		Receiver:          receiver.Receiver,
		Status:            receiver.Status,
		GroupLabels:       receiver.GroupLabels,
		CommonLabels:      receiver.CommonLabels,
		CommonAnnotations: receiver.CommonAnnotations,
		ExternalURL:       receiver.ExternalURL,
		Version:           receiver.Version,
		GroupKey:          receiver.GroupKey,
		TruncatedAlerts:   receiver.TruncatedAlerts,
	}
}

type HandleAggregationSendResult struct {
	FiringErr      error
	ResolvedErr    error
	FiringAlerts   []*Alert
	ResolvedAlerts []*Alert
}

type NormalSendResult struct {
	Alert   *Alert
	SendErr error
}
