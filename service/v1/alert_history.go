package v1

import (
	"context"
	"fmt"

	"github.com/qinquanliuxiang666/alertmanager/base/helper"
	"github.com/qinquanliuxiang666/alertmanager/base/types"
	"github.com/qinquanliuxiang666/alertmanager/model"
	"github.com/qinquanliuxiang666/alertmanager/store"
)

type AlertHistoryServicer interface {
	QueryHistory(ctx context.Context, req *types.IDRequest) (*model.AlertHistory, error)
	ListHistory(ctx context.Context, req *types.AlertHistoryListRequest) (*types.AlertHistoryListResponse, error)
}

type alertHistoryService struct {
	cache store.CacheStorer
}

func NewHistoryServicer(cache store.CacheStorer) AlertHistoryServicer {
	return &alertHistoryService{}
}

func (recevicer *alertHistoryService) QueryHistory(ctx context.Context, req *types.IDRequest) (*model.AlertHistory, error) {
	return al.WithContext(ctx).Where(al.ID.Eq(int(req.ID))).First()
}

func (recevicer *alertHistoryService) ListHistory(ctx context.Context, req *types.AlertHistoryListRequest) (*types.AlertHistoryListResponse, error) {
	var (
		alertAlertHistorys []*model.AlertHistory
		total              int64
		sql                = al.WithContext(ctx)
		err                error
	)

	if req.AlertName != "" {
		sql = sql.Where(al.Alertname.Like(req.AlertName + "%"))
	} else if req.Fingerprint != "" {
		sql.Where(al.Fingerprint.Like(req.Fingerprint + "%"))
	} else if req.Severity != "" {
		sql.Where(al.Severity.Eq(req.Severity))
	} else if req.Instance != "" {
		sql.Where(al.Instance.Eq(req.Instance))
	}

	if req.StartsAt != nil {
		sql.Where(al.StartsAt.Gt(*req.StartsAt))
	} else if req.EndsAt != nil {
		sql.Where(al.EndsAt.Gt(*req.EndsAt))
	}

	if total, err = sql.Count(); err != nil {
		return nil, err
	}

	if req.Sort != "" && req.Direction != "" {
		sort, ok := al.GetFieldByName(req.Sort)
		if !ok {
			return nil, fmt.Errorf("invalid sort field: %s", req.Sort)
		}
		sql = sql.Order(helper.Sort(sort, req.Direction))
	}

	if req.PageSize == 0 || req.Page == 0 {
		if alertAlertHistorys, err = sql.Find(); err != nil {
			return nil, err
		}
	} else {
		if alertAlertHistorys, err = sql.Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Find(); err != nil {
			return nil, err
		}
	}
	return types.NewAlertHistoryListResponse(alertAlertHistorys, total, req.PageSize, req.Page), nil
}
