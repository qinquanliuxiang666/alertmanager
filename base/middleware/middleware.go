package middleware

import (
	"github.com/gin-gonic/gin"
	apitypes "github.com/qinquanliuxiang666/alertmanager/base/types"
	"github.com/qinquanliuxiang666/alertmanager/pkg/casbin"
	"github.com/qinquanliuxiang666/alertmanager/pkg/jwt"
	"github.com/qinquanliuxiang666/alertmanager/store"
)

type MiddlewareInterface interface {
	Auth() gin.HandlerFunc
	AuthZ() gin.HandlerFunc
	Session() gin.HandlerFunc
}

type Middleware struct {
	jwtImpl   jwt.JwtInterface
	authZImpl casbin.AuthChecker
	cacheImpl store.CacheStorer
}

func NewMiddleware(jwtImpl jwt.JwtInterface, authZImpl casbin.AuthChecker, cacheImpl store.CacheStorer) *Middleware {
	return &Middleware{
		jwtImpl:   jwtImpl,
		authZImpl: authZImpl,
		cacheImpl: cacheImpl,
	}
}

func (m *Middleware) Abort(c *gin.Context, code int, err error) {
	c.JSON(code, apitypes.NewResponseWithOpts(code, apitypes.WithError(err.Error())))
	c.Error(err)
	c.Abort()
}
