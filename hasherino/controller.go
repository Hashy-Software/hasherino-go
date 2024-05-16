package hasherino

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Controlls everything in the app. Called by UI code, making it UI library agnostic.
type HasherinoController struct {
	appId       string
	twitchOAuth *TwitchOAuth
	chatWS      *TwitchChatWebsocket
	memDB       *gorm.DB
	permDB      *gorm.DB
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
	c := &HasherinoController{
		appId:       "hvmj7blkwy2gw3xf820n47i85g4sub",
		twitchOAuth: NewTwitchOAuth(),
		chatWS:      chatWS,
		memDB:       memDB,
		permDB:      permDB,
	}
	go c.twitchOAuth.ListenForOAuthRedirect(c)
	return c, nil
}

func (hc *HasherinoController) AddAccount(id string, login string, token string) error {
	return hc.permDB.Transaction(func(tx *gorm.DB) error {
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

func (hc *HasherinoController) RemoveAccount(id string) error {
	return hc.permDB.Delete(&Account{}, "Id = ?", id).Error
}

func (hc *HasherinoController) GetAccounts() ([]*Account, error) {
	accounts := []*Account{}
	result := hc.permDB.Find(&accounts)
	return accounts, result.Error
}

func (hc *HasherinoController) SetActiveAccount(id string) error {
	return hc.permDB.Transaction(func(tx *gorm.DB) error {
		// Disable every active account(should be just one, but just in case)
		activeAccounts := []*Account{}
		result := tx.Find(&activeAccounts, "Active = ?", true)
		if result.Error != nil {
			return result.Error
		}
		if len(activeAccounts) > 0 {
			for _, account := range activeAccounts {
				account.Active = false
				result = tx.Save(&account)
				if result.Error != nil {
					return result.Error
				}
			}
		}
		// Set account as active
		account := &Account{}
		result = tx.Take(&account, "Id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		account.Active = true
		result = tx.Save(&account)

		// Commit transation
		return nil
	})
}

func (hc *HasherinoController) GetActiveAccount() (*Account, error) {
	account := &Account{}
	result := hc.permDB.Take(&account, "Active = ?", true)

	if result.Error != nil {
		return nil, result.Error
	}

	return account, nil
}

func (hc *HasherinoController) OpenOAuthPage() {
	hc.twitchOAuth.OpenOAuthPage(hc.appId)
}
