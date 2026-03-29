package helper

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/qinquanliuxiang666/alertmanager/model"
)

func VerificationAlertFeishuConfig(channel *model.AlertChannel) (appid, appSecret string, err error) {
	var ok bool
	config := make(map[string]any, 0)
	err = json.Unmarshal(channel.Config, &config)
	if err != nil {
		return "", "", fmt.Errorf("验证飞书客户端配置失败, %s", err)
	}
	if err := VerificationAlertConfig(channel.Name, model.ChannelTypeFeishuApp, config); err != nil {
		return "", "", fmt.Errorf("验证飞书客户端配置失败, %s", err)
	}
	_appID := config["app_id"]
	appid, ok = _appID.(string)
	if !ok {
		return "", "", fmt.Errorf("获取飞书 app_id 置失败, %s", err)
	}
	_appSecret := config["app_secret"]
	appSecret, ok = _appSecret.(string)
	if !ok {
		return "", "", fmt.Errorf("获取飞书 app_secret 置失败, %s", err)
	}
	return appid, appSecret, nil
}

func VerificationAlertConfig(channelName string, channelType model.ChannelType, config map[string]any) error {
	switch channelType {
	case model.ChannelTypeFeishuApp:
		appID := config["app_id"]
		appSecret := config["app_secret"]
		receiveId := config["receive_id"]
		receiveIdType := config["receive_id_type"]
		if appID == nil {
			return fmt.Errorf("alertChannel.Config 飞书应用 ID 不存在")
		}
		if appSecret == nil {
			return fmt.Errorf("alertChannel.Config 飞书应用 secret 不存在")
		}
		if receiveId == nil {
			return fmt.Errorf("alertChannel.Config 飞书应用 receiveId 不存在")
		}
		if receiveIdType == nil {
			return fmt.Errorf("alertChannel.Config 飞书应用 receiveIdType 不存在")
		}
		return nil
	default:
		return fmt.Errorf("%s 告警是不支持的告警类型 %s", channelName, channelType)
	}
}

func GetAlertMapKey(fingerprint string, startAt time.Time) string {
	return fmt.Sprintf("%s-%d", fingerprint, startAt.UnixNano())
}
