package store_test

import (
	"github.com/qinquanliuxiang666/alertmanager/base/conf"
	"github.com/qinquanliuxiang666/alertmanager/base/data"
	"github.com/qinquanliuxiang666/alertmanager/base/log"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func init() {
	var err error
	conf.LoadConfig("../../config.yaml")
	db, _, err = data.NewDB()
	if err != nil {
		panic(err)
	}
	log.NewLogger()
}
