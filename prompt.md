# prompt

```go
package v1

import (
	"context"

	"github.com/qinquanliuxiang666/alertmanager/base/types"
	"github.com/qinquanliuxiang666/alertmanager/model"
)

type AlertTemplateServicer interface {
	CreateApi(ctx context.Context, req *types.ApiCreateRequest) error
	UpdateApi(ctx context.Context, req *types.ApiUpdateRequest) error
	DeleteApi(ctx context.Context, req *types.IDRequest) error
	QueryApi(ctx context.Context, req *types.IDRequest) (*model.Api, error)
	ListApi(ctx context.Context, pagination *types.ApiListRequest) (*types.ApiListResponse, error)
}

type AlertTemplateService struct{}

func AlertTemplateServicer() AlertTemplateServicer {
	return &AlertTemplateService{}
}
```

vscode 如何快速实现这些方法

在 VS Code 中，如果你安装了官方的 Go 扩展 (Go Team at Google)，有几种非常快的方法可以自动生成接口实现。

使用 Go: Generate Interface Stubs 命令（最推荐）, 这是最标准的方法，可以一次性生成所有方法的空壳。

按下键盘快捷键：Ctrl + Shift + P (Mac 为 Cmd + Shift + P)。输入并选择：Go: Generate Interface Stubs。在弹出的输入框中按照以下格式输入：

```go
recevicer *alertChannelService AlertChannelServicer
```

按下回车，所有方法就会自动插入到文件中

## base64

```go
type AlertTemplateCreateRequest struct {
	Name                string `json:"name" binding:"required"`
	Description         string `json:"description"`
	Template            string `json:"template" binding:"required"`
	AggregationTemplate string `json:"aggregationTemplate"`
	AlertChannelID      int    `json:"alertChannelID"`
}
```

Template AggregationTemplate 设置base64验证

## json tag

```go
type AlertChannelCreateRequest struct {
	Name              string `json:"name" binding:"required"`
	Type              string `json:"type" binding:"required"`
	Status            int    `json:"status" binding:"required"`
	AggregationStatus int    `json:"aggregationStatus" binding:"required"`
	Config            any    `json:"config" binding:"required"`
	Description       string `json:"description"`
}
```

- Name 限制长度15
- type 为 feishuApp或feishuBoot或webhook
- Status 0 或 1
- AggregationStatus 0 或 1
- Config 对象

## 缓存和数据库一致性

```go
func (recevicer *alertChannelService) UpdateChannel(ctx context.Context, req *types.AlertChannelUpdateRequest) error {
	var update bool
	sql := ac.WithContext(ctx)

	acObj, err := sql.Where(ac.ID.Eq(int(req.ID))).First()
	if err != nil {
		return err
	}

	acObj.Type = model.ChannelType(req.Type)
	acObj.Status = model.ChannelStatus(req.Status)
	acObj.AggregationStatus = model.AggregationStatus(req.AggregationStatus)

	if err := helper.VerificationAlertConfig(acObj.Name, model.ChannelType(req.Type), req.Config); err != nil {
		return err
	}

	c, err := json.Marshal(req.Config)
	if err != nil {
		return err
	}
	acObj.Config = c
	acObj.Description = req.Description
	if acObj.AlertTemplateID != req.TemplateID {
		update = true
	}
	acObj.AlertTemplateID = req.TemplateID

	store.Q.Transaction(func(tx *store.Query) error {

		if update {
			if err := recevicer.cache.DelKey(ctx, store.AlertType, acObj.Name); err != nil {
				return err
			}

			if err := recevicer.cache.SetObject(ctx, store.AlertType, acObj.Name, acObj, store.NeverExpires); err != nil {
				return err
			}
		}

		return tx.AlertChannel.WithContext(ctx).Save(acObj)
	})

	return nil
}
```

如何保证缓存和数据库数据一致性
