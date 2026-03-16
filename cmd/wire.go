// go:build wireinject
//go:build wireinject
// +build wireinject

package cmd

import (
	"github.com/google/wire"
	"github.com/qinquanliuxiang666/alertmanager/base/app"
	"github.com/qinquanliuxiang666/alertmanager/base/data"
	"github.com/qinquanliuxiang666/alertmanager/base/middleware"
	"github.com/qinquanliuxiang666/alertmanager/base/router"
	"github.com/qinquanliuxiang666/alertmanager/base/server"
	"github.com/qinquanliuxiang666/alertmanager/controller"
	"github.com/qinquanliuxiang666/alertmanager/pkg"
	"github.com/qinquanliuxiang666/alertmanager/service"
	"github.com/qinquanliuxiang666/alertmanager/store"
)

func InitApplication() (*app.Application, func(), error) {
	panic(wire.Build(
		data.DataProviderSet,
		pkg.PkgProviderSet,
		store.StoreProviderSet,
		service.ServiceProviderSet,
		controller.ControllerProviderSet,
		middleware.MiddlewareProviderSet,
		router.RouterProviderSet,
		server.ServerProviderSet,
		app.AppProviderSet,
	))
}
