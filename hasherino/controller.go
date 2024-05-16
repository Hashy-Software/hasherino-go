package hasherino

import (
	"errors"
	"log"

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
	permDB.AutoMigrate(&Account{}, &Tab{})
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

func (hc *HasherinoController) AddTab(channel string) error {
	err := hc.permDB.Transaction(func(tx *gorm.DB) error {
		activeAccount := &Account{}
		result := hc.permDB.Take(&activeAccount, "Active = ?", true)
		if result.Error != nil {
			return errors.New("No active account")
		}

		helix := NewHelix(hc.appId)
		users, err := helix.GetUsers(activeAccount.Token, []string{channel})
		if err != nil || len(users.Data) != 1 {
			return errors.New("Failed to obtain channel's id and login")
		}

		tab := &Tab{
			Id:          users.Data[0].ID,
			Login:       users.Data[0].Login,
			DisplayName: users.Data[0].DisplayName,
			Selected:    false,
		}

		result = tx.Create(&tab)

		err = hc.chatWS.Join(channel)
		if err != nil {
			log.Printf("Failed to join channel %s: %s", channel, err)
			return errors.New("Failed to join channel " + channel)
		}

		return nil
	})
	return err
}

func (hc *HasherinoController) RemoveTab(id string) error {
	err := hc.permDB.Transaction(func(tx *gorm.DB) error {
		tab := &Tab{}
		result := hc.permDB.Take(&tab, "Id = ?", id)
		if result.Error != nil {
			return errors.New("Tab not found for id " + id)
		}

		err := hc.chatWS.Part(tab.Login)
		if err != nil {
			return errors.New("Failed to part channel" + tab.Login)
		}

		return nil
	})
	return err
}

func (hc *HasherinoController) GetTabs() ([]*Tab, error) {
	tabs := []*Tab{}
	result := hc.permDB.Find(&tabs)
	return tabs, result.Error
}

func (hc *HasherinoController) Listen(callback func(string)) error {
	// TODO: parse string here and call each tab's callback with a parsed message object(take a channel-callback map)
	// Try to find an existing IRC parser
	activeAccount, err := hc.GetActiveAccount()
	if err != nil {
		return err
	}
	if hc.chatWS == nil || hc.chatWS.State == Disconnected {
		hc.chatWS, err = hc.chatWS.New(activeAccount.Token, activeAccount.Login)
		if err != nil {
			return err
		}
		err = hc.chatWS.Connect()
		if err != nil {
			return err
		}
	}

	go hc.chatWS.Listen(callback)
	return nil
}
