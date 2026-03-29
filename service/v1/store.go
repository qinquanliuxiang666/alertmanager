package v1

import (
	"github.com/qinquanliuxiang666/alertmanager/store"
)

var (
	u      = store.User
	r      = store.Role
	a      = store.Api
	c      = store.CasbinRule
	oauth2 = store.Oauth2User
	al     = store.AlertHistory
	ac     = store.AlertChannel
	at     = store.AlertTemplate
	as     = store.AlertSendRecord
)

func NewStore() {
	u = store.User
	r = store.Role
	a = store.Api
	c = store.CasbinRule
	oauth2 = store.Oauth2User
	al = store.AlertHistory
	ac = store.AlertChannel
	at = store.AlertTemplate
	as = store.AlertSendRecord
}
