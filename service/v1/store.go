package v1

import (
	"github.com/qinquanliuxiang666/alertmanager/store"
)

var (
	u         = store.User
	r         = store.Role
	a         = store.Api
	c         = store.CasbinRule
	oauth2    = store.Oauth2User
	aHistory  = store.AlertHistory
	aChannel  = store.AlertChannel
	aTemlpate = store.AlertTemplate
	aSend     = store.AlertSendRecord
	aSilence  = store.AlertSilence
)

func NewStore() {
	u = store.User
	r = store.Role
	a = store.Api
	c = store.CasbinRule
	oauth2 = store.Oauth2User
	aHistory = store.AlertHistory
	aChannel = store.AlertChannel
	aTemlpate = store.AlertTemplate
	aSend = store.AlertSendRecord
	aSilence = store.AlertSilence
}
