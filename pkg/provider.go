package pkg

import (
	"github.com/google/wire"
	"github.com/qinquanliuxiang666/alertmanager/pkg/alert"
	"github.com/qinquanliuxiang666/alertmanager/pkg/casbin"
	"github.com/qinquanliuxiang666/alertmanager/pkg/feishu"
	"github.com/qinquanliuxiang666/alertmanager/pkg/jwt"
	localcache "github.com/qinquanliuxiang666/alertmanager/pkg/local_cache"
	"github.com/qinquanliuxiang666/alertmanager/pkg/oauth"
)

var PkgProviderSet = wire.NewSet(
	wire.Bind(new(jwt.JwtInterface), new(*jwt.GenerateToken)),
	jwt.NewGenerateToken,
	alert.NewAlertUtiler,
	feishu.NewFeiShu,
	casbin.NewEnforcer,
	casbin.NewCasbinManager,
	casbin.NewAuthChecker,
	oauth.NewOAuth2,
	localcache.NewCacher,
)
