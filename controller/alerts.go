package controller

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/qinquanliuxiang666/alertmanager/service/v1"
)

type AlertManagerController interface {
	ReceiveAlerts(c *gin.Context)
}

type alertManagerController struct {
	alertService v1.AlertsServicer
}

func NewAlertManagerController(alertService v1.AlertsServicer) AlertManagerController {
	return &alertManagerController{
		alertService: alertService,
	}
}

func (receiver *alertManagerController) ReceiveAlerts(c *gin.Context) {
	ResponseOnlySuccess(c, receiver.alertService.SendAlert, bindTypeQuery, bindTypeJson)
	// req := new(types.AlertReceiveReq)
	// if err := c.ShouldBind(&req); err != nil {
	// 	zap.S().Errorf("get playload filaed, err: %v", err)
	// 	c.JSON(500, err)
	// 	return
	// }
	// d, err := json.Marshal(&req)
	// if err != nil {
	// 	zap.S().Errorf("get palyload filaed, err: %v", err)
	// 	c.JSON(500, err)
	// 	return
	// }
	// fmt.Println("☀️------------------------------------☀️")
	// fmt.Println(string(d))
	// fmt.Println("🌙------------------------------------🌙")
	// c.JSON(200, req)
}
