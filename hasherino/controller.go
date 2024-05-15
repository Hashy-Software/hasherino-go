package hasherino

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Controlls everything in the app. Called by UI code, making it UI library agnostic.
type HasherinoController struct {
	chatWS *TwitchChatWebsocket
	memDB  *gorm.DB
	permDB *gorm.DB
}

func (hc *HasherinoController) New() (*HasherinoController, error) {
	chatWS := &TwitchChatWebsocket{}
	memDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	permDB, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	permDB.AutoMigrate(&Account{})
	return &HasherinoController{chatWS: chatWS, memDB: memDB, permDB: permDB}, nil
}

func (hc *HasherinoController) AddAccount(id string, login string, token string) error {
	return hc.memDB.Transaction(func(tx *gorm.DB) error {
		// Get one account, no specific order
		result := tx.Take(&Account{})
		// No account exists, so the first one should be active
		active := result.Error != nil
		result = tx.Create(&Account{Id: id, Login: login, Token: token, Active: active})
		if result.Error != nil {
			return errors.New("could not create account")
		}
		// no errors, commit transaction
		return nil
	})
}
