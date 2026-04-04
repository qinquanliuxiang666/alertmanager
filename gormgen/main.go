package main

import (
	"github.com/qinquanliuxiang666/alertmanager/base/conf"
	"github.com/qinquanliuxiang666/alertmanager/base/data"
	"github.com/qinquanliuxiang666/alertmanager/model"
	"gorm.io/gen"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "./store",
		Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	conf.LoadConfig("./config.yaml")
	db, clear, err := data.NewDB()
	if err != nil {
		panic(err)
	}
	defer clear()
	db.AutoMigrate(model.AlertHistory{}, model.AlertChannel{}, model.AlertTemplate{}, model.AlertSendRecord{}, &model.AlertSilence{})
	g.UseDB(db)
	g.ApplyBasic(model.User{}, model.Role{}, model.Api{}, model.CasbinRule{}, model.Oauth2User{}, model.AlertHistory{}, model.AlertChannel{}, model.AlertTemplate{}, model.AlertSendRecord{}, model.AlertSilence{})
	g.Execute()
}
