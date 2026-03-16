package store_test

import (
	"context"
	"testing"

	"github.com/qinquanliuxiang666/alertmanager/model"
	"github.com/qinquanliuxiang666/alertmanager/store"
)

func TestCreateTable(t *testing.T) {
	if err := db.AutoMigrate(&model.Oauth2User{}); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
}

func TestStoreQ(t *testing.T) {
	err := store.Q.Transaction(func(tx *store.Query) error {
		tx.WithContext(context.Background()).AlertSendRecord.Create(&model.AlertSendRecord{
			ID:         2,
			SendStatus: "sss",
		})
		tx.WithContext(context.Background()).AlertSendRecord.Create(&model.AlertSendRecord{
			ID:         1,
			SendStatus: "sss",
		})
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}
