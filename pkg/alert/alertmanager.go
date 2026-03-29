package alert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/qinquanliuxiang666/alertmanager/base/constant"
	"github.com/qinquanliuxiang666/alertmanager/base/log"
	"github.com/qinquanliuxiang666/alertmanager/base/types"
	"github.com/qinquanliuxiang666/alertmanager/model"
	"github.com/qinquanliuxiang666/alertmanager/store"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AlertUtiler interface {
	AlertGroup(ctx context.Context, alerts []*types.Alert) (firingAlerts, resolvedAlerts []*types.Alert)
	SaveAggregationAlert(ctx context.Context, alertChannel *model.AlertChannel, req *types.HandleAggregationSendResult)
	SaveNormalAlerts(ctx context.Context, alertChannel *model.AlertChannel, results []*types.NormalSendResult)
}

type AlertUtil struct {
}

func NewAlertUtiler() AlertUtiler {
	return &AlertUtil{}
}

func (receiver *AlertUtil) AlertGroup(ctx context.Context, alerts []*types.Alert) (firingAlerts, resolvedAlerts []*types.Alert) {
	if len(alerts) == 0 {
		return
	}

	firingAlerts = make([]*types.Alert, 0)
	resolvedAlerts = make([]*types.Alert, 0)
	for _, alert := range alerts {
		if alert.Status == constant.AlertStatusFiring {
			// 告警逻辑
			// TODO 判断告警静默,如果静默continue
			silence, err := receiver.getSilence(ctx, alert)
			if err != nil {
				zap.L().Error("判断静默失败", zap.Error(err))
				continue
			}

			if silence {
				continue
			}

			firingAlerts = append(firingAlerts, alert)
		} else {
			resolvedAlerts = append(resolvedAlerts, alert)
		}
	}
	return
}

// TODO getSilence
func (receiver *AlertUtil) getSilence(ctx context.Context, alert *types.Alert) (silence bool, err error) {
	return silence, nil
}

func (receiver *AlertUtil) SaveAggregationAlert(ctx context.Context, alertChannel *model.AlertChannel, req *types.HandleAggregationSendResult) {
	// d, _ := json.Marshal(alertChannel)
	// dd, _ := json.Marshal(req)
	// fmt.Println("☀️------------------------------------☀️")
	// fmt.Println(string(d))
	// fmt.Println("🌙------------------------------------🌙")
	// fmt.Println("☀️------------------------------------☀️")
	// fmt.Println(string(dd))
	// fmt.Println("🌙------------------------------------🌙")

	// 1. 处理 Firing 告警
	if len(req.FiringAlerts) > 0 {
		// 将未恢复告警结束时间设置为 nil
		for i := range req.FiringAlerts {
			req.FiringAlerts[i].EndsAt = nil
		}

		if err := receiver.processAlertGroup(ctx, alertChannel, req.FiringAlerts, req.FiringErr); err != nil {
			log.WithRequestID(ctx).Error("处理 Firing 告警落库失败", zap.Error(err))
		}
	}

	// 2. 处理 Resolved 告警
	if len(req.ResolvedAlerts) > 0 {
		if err := receiver.processAlertGroup(ctx, alertChannel, req.ResolvedAlerts, req.ResolvedErr); err != nil {
			log.WithRequestID(ctx).Error("处理 Resolved 告警落库失败", zap.Error(err))
		}
	}

}

// processAlertGroup 提取出的公用处理逻辑
func (receiver *AlertUtil) processAlertGroup(ctx context.Context, alertChannel *model.AlertChannel, alerts []*types.Alert, sendErr error) error {
	alerthistoryStore := store.AlertHistory
	alertSendRecordStore := store.AlertSendRecord
	firstAlert := alerts[0]
	searchTime := firstAlert.StartsAt.Truncate(time.Millisecond)
	// 1. 尝试查询记录
	firstAlertHistory, err := alerthistoryStore.WithContext(ctx).
		Preload(alerthistoryStore.AlertSendRecord).
		Where(alerthistoryStore.Fingerprint.Eq(firstAlert.Fingerprint)).
		Where(alerthistoryStore.StartsAt.Eq(searchTime)).
		First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Error("查询firstAlertHistory记录失败, %s", zap.Error(err))
		return err
	}

	// 情况 A: 已经有记录了，找到对应的 SendRecord 并更新状态
	if firstAlertHistory != nil {
		if firstAlertHistory.AlertSendRecordID == 0 {
			return fmt.Errorf("firstAlertHistory.AlertSendRecordID 等于 0, 无法关联记录")
		}

		// 如果记录已存在，且有外键，则更新发送记录
		record, err := alertSendRecordStore.WithContext(ctx).Where(alertSendRecordStore.ID.Eq(firstAlertHistory.AlertSendRecordID)).First()
		if err != nil {
			return fmt.Errorf("")
		}
		updateStatus(firstAlertHistory.AlertSendRecord, sendErr)
		if err := alertSendRecordStore.WithContext(ctx).Save(record); err != nil {
			return fmt.Errorf("更新 AlertSendRecord 失败, %v", err)
		}

		go func(reqIDCtx context.Context, alertsData []*types.Alert) {
			defer func() {
				if r := recover(); r != nil {
					log.WithRequestID(reqIDCtx).Error("异步更新告警历史记录发生崩溃(Panic)", zap.Any("error", r))
				}
			}()

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// --- 优化点 1: 构造批量查询条件 ---
			// 构造类似 WHERE (fingerprint, starts_at) IN (('fp1', 't1'), ('fp2', 't2'))
			var queryArgs [][]interface{}
			for _, a := range alertsData {
				queryArgs = append(queryArgs, []interface{}{
					a.Fingerprint,
					a.StartsAt.Truncate(time.Millisecond),
				})
			}

			// --- 优化点 2: 一次性捞出所有匹配的记录 ---
			var existingHistories []*model.AlertHistory
			err = alerthistoryStore.WithContext(timeoutCtx).
				UnderlyingDB(). // 切换到原生 gorm 以支持元组查询
				Where("(fingerprint, starts_at) IN ?", queryArgs).
				Find(&existingHistories).Error

			if err != nil {
				log.WithRequestID(reqIDCtx).Error("批量查询告警历史失败", zap.Error(err))
				return
			}

			if len(existingHistories) == 0 {
				return
			}

			// --- 优化点 3: 内存匹配 ---
			// 将查出的结果转成 map 方便快速比对
			historyMap := make(map[string]*model.AlertHistory)
			// 全部没有恢复的告警
			noResolveDMap := receiver.disableBefoceAlertHistory(timeoutCtx, firstAlert.Labels["cluster"])
			// 本次告警包含的未恢复的告警
			exitsHistoryMap := make(map[string][]*model.AlertHistory, len(noResolveDMap))
			endTime := time.Now().Local().Truncate(time.Millisecond)
			for exisIndex := range existingHistories {
				currAlert := existingHistories[exisIndex]
				oldAlertHistorys, ok := noResolveDMap[currAlert.Fingerprint]
				if ok {
					for i := range oldAlertHistorys {
						// 相同集群、相同告警指纹，如果存在为结束的告警，但是现在又触发了一个，那么便将早的告警结束掉
						if oldAlertHistorys[i].StartsAt.Before(currAlert.StartsAt) {
							oldAlertHistorys[i].Status = constant.AlertStatusResolved
							oldAlertHistorys[i].EndsAt = &endTime
							exitsHistoryMap[currAlert.Fingerprint] = append(exitsHistoryMap[currAlert.Fingerprint], oldAlertHistorys[i])
						}
					}
				}
				// key 使用指纹 + 时间戳(纳秒)
				key := fmt.Sprintf("%s-%d", currAlert.Fingerprint, currAlert.StartsAt.UnixNano())
				historyMap[key] = currAlert
			}

			objs := make([]*model.AlertHistory, 0)
			for _, a := range alertsData {
				key := fmt.Sprintf("%s-%d", a.Fingerprint, a.StartsAt.Truncate(time.Millisecond).UnixNano())
				if alertObj, ok := historyMap[key]; ok {
					if a.Status == constant.AlertStatusResolved {
						alertObj.Status = constant.AlertStatusResolved
						alertObj.EndsAt = a.EndsAt
					}
					alertObj.SendCount += 1
					objs = append(objs, alertObj)
				}
			}

			// --- 批量保存 ---
			if len(objs) > 0 {
				if err := alerthistoryStore.WithContext(timeoutCtx).Save(objs...); err != nil {
					log.WithRequestID(reqIDCtx).Error("异步批量更新 alerthistory 失败", zap.Error(err))
				}
			}

			if len(exitsHistoryMap) > 0 {
				exitsHistorySline := make([]*model.AlertHistory, 0, len(exitsHistoryMap)*2)
				for i := range exitsHistoryMap {
					exitsHistorySline = append(exitsHistorySline, exitsHistoryMap[i]...)
				}
				if err := alerthistoryStore.WithContext(timeoutCtx).Save(exitsHistorySline...); err != nil {
					log.WithRequestID(reqIDCtx).Error("异步批量更新过期告警 alerthistory.status 失败", zap.Error(err))
				}
			}
		}(ctx, alerts)

		return nil
	}

	// 情况 B: 没有记录，手动处理外键关联
	return store.Q.Transaction(func(tx *store.Query) error {
		ahStore := tx.AlertHistory
		asStore := tx.AlertSendRecord
		// 1. 先创建 AlertSendRecord
		sendRecord := &model.AlertSendRecord{}
		updateStatus(sendRecord, sendErr)
		// 必须先 Create，这样 GORM 才会把自增 ID 填充回 sendRecord.ID
		if err := asStore.WithContext(ctx).Create(sendRecord); err != nil {
			return fmt.Errorf("创建 AlertSendRecord 失败, %s", err)
		}
		// 2. 准备 AlertHistory 数据，并将刚才获取到的 ID 赋值给外键
		historyModels := make([]*model.AlertHistory, 0, len(alerts))
		for _, a := range alerts {
			h, err := convertToModel(a, alertChannel.ID)
			if err != nil {
				return err
			}
			// 【关键点】手动关联外键 ID
			h.AlertSendRecordID = sendRecord.ID
			historyModels = append(historyModels, h)
		}
		// 3. 批量创建 History 记录
		return ahStore.WithContext(ctx).Create(historyModels...)
	})
}

// 辅助函数：更新状态信息
func updateStatus(record *model.AlertSendRecord, sendErr error) {
	if sendErr == nil {
		record.SendStatus = model.AlertSendRecordStatusSuccess
	} else {
		record.SendStatus = model.AlertSendRecordStatusFailed
		record.ErrorMessage = fmt.Sprintf("record.ErrorMessage \n%s", sendErr.Error())
	}
}

// 辅助函数：将业务 Alert 转换为 DB Model
func convertToModel(a *types.Alert, channelID int) (*model.AlertHistory, error) {
	labelByte, _ := json.Marshal(a.Labels)
	annotationsByte, _ := json.Marshal(a.Annotations)
	return &model.AlertHistory{
		Fingerprint:    a.Fingerprint,
		StartsAt:       a.StartsAt,
		Cluster:        a.Labels["cluster"],
		EndsAt:         a.EndsAt,
		Status:         a.Status,
		Alertname:      a.Labels["alertname"],
		Severity:       a.Labels["severity"],
		Instance:       a.Labels["instance"],
		Labels:         datatypes.JSON(labelByte),
		Annotations:    datatypes.JSON(annotationsByte),
		AlertChannelID: channelID,
		SendCount:      1,
	}, nil
}

func (receiver *AlertUtil) SaveNormalAlerts(ctx context.Context, alertChannel *model.AlertChannel, results []*types.NormalSendResult) {
	if len(results) == 0 {
		return
	}
	go func() {
		dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, res := range results {
			// 将单条告警包装成切片，复用你之前写的 processAlertGroup
			// 注意：这里需要根据 Status 分别对应 FiringErr 或 ResolvedErr
			alertSlice := []*types.Alert{res.Alert}

			if err := receiver.processAlertGroup(dbCtx, alertChannel, alertSlice, res.SendErr); err != nil {
				zap.L().Error("单条告警落库失败",
					zap.String("fingerprint", res.Alert.Fingerprint),
					zap.Error(err),
				)
			}
		}
	}()
}

func (receiver *AlertUtil) disableBefoceAlertHistory(ctx context.Context, cluster string) map[string][]*model.AlertHistory {
	al := store.AlertHistory.WithContext(ctx)

	alertHistorys, err := al.Where(store.AlertHistory.Cluster.Eq(cluster)).Where(store.AlertHistory.EndsAt.IsNull()).Find()
	if err != nil {
		zap.L().Error("查询旧的 alertHistory 失败", zap.Error(err))
		return nil
	}
	alertHistoryMap := make(map[string][]*model.AlertHistory, len(alertHistorys))
	for i := range alertHistorys {
		fp := alertHistorys[i].Fingerprint
		alertHistoryMap[fp] = append(alertHistoryMap[fp], alertHistorys[i])
	}

	return alertHistoryMap
}
