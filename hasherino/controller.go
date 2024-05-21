package hasherino

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Controlls everything in the app. Called by UI code, making it UI library agnostic.
type HasherinoController struct {
	appId       string
	selectedTab string
	callbackMap map[string]func(ChatMessage)
	twitchOAuth *TwitchOAuth
	chatWS      *TwitchChatWebsocket
	memDB       *gorm.DB
	permDB      *gorm.DB
}

func (hc *HasherinoController) New(callbackMap map[string]func(ChatMessage)) (*HasherinoController, error) {
	chatWS := &TwitchChatWebsocket{}

	dataFolder, err := GetDataFolder()
	if err != nil {
		return nil, err
	}

	memDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	memDB.AutoMigrate(&TempTab{}, &ChatUser{}, &Emote{}, &ChatUserTempTab{})

	permDB, err := gorm.Open(sqlite.Open(filepath.Join(dataFolder, "gorm.db")), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	permDB.AutoMigrate(&Account{}, &Tab{}, &AppSettings{})

	c := &HasherinoController{
		appId:       "hvmj7blkwy2gw3xf820n47i85g4sub",
		callbackMap: callbackMap,
		twitchOAuth: NewTwitchOAuth(),
		chatWS:      chatWS,
		memDB:       memDB,
		permDB:      permDB,
	}
	settings := &AppSettings{}
	result := permDB.Take(settings)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = permDB.Create(&AppSettings{ChatMessageLimit: 100, ChatHistory: false})
	} else if result.Error != nil {
		return nil, err
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
	return hc.permDB.Transaction(func(tx *gorm.DB) error {
		account := &Account{}
		result := tx.Take(&account, "Id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if account.Active {
			hc.chatWS.Close()
		}
		result = tx.Delete(&account)
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
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

		if result.Error != nil {
			return result.Error
		}

		if hc.chatWS != nil {
			hc.chatWS.Close()
		}
		go hc.Listen()

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

		_, exists := hc.chatWS.channels[channel]
		if exists {
			return errors.New("Already joined channel " + channel)
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

		err = hc.AddTempTabs(&[]string{users.Data[0].ID})

		if err != nil {
			return err
		}

		return nil
	})
	return err
}

func (hc *HasherinoController) AddTempTabs(channelIds *[]string) error {
	go func(channelIds *[]string) {
		err := hc.memDB.Transaction(func(tx *gorm.DB) error {
			emotes, err, isCached := STVGetGlobalEmotes()
			// Insert global emotes into tempDB
			if !isCached {
				if err != nil {
					log.Printf("Failed to load global 7tv emotes: %s", err)
				} else {
					emoteObjs := []Emote{}
					for _, emote := range emotes.Data.EmoteSet.Emotes {
						tempFile, err := os.CreateTemp("", "")
						if err != nil {
							log.Printf("Failed to create temp file %s for 7tv emote %s", err, emote.Name)
							continue
						}
						emoteObjs = append(emoteObjs, Emote{
							Id:        emote.ID,
							Source:    SevenTV,
							Name:      emote.Name,
							Animated:  emote.Data.Animated,
							ChannelID: nil,
							OwnerID:   "",
							Owner:     nil,
							TempFile:  tempFile.Name(),
						})
						tempFile.Close()
					}
					log.Println("Loaded " + strconv.Itoa(len(emoteObjs)) + " 7tv global emotes")
					result := tx.Create(&emoteObjs)
					if result.Error != nil {
						log.Printf("Failed to save global 7tv emotes: %s", result.Error)
					}
				}
			}

			// Insert channel 7tv emotes
			stvUsers := make(map[string]*STVUserJson)
			mutex := sync.Mutex{}
			wg := sync.WaitGroup{}
			for _, channelId := range *channelIds {
				wg.Add(1)
				go func(channelId string) {
					defer wg.Done()
					stvUser, err := STVGetUser(channelId)
					if err != nil {
						log.Printf("Failed to load 7tv user emotes: %s", err)
					} else {
						mutex.Lock()
						stvUsers[channelId] = stvUser
						mutex.Unlock()
					}
				}(channelId)
			}
			wg.Wait()

			if len(stvUsers) > 0 {
				for channelId, stvUser := range stvUsers {
					if channelId == "" {
						continue
					}

					emoteObjs := []Emote{}
					if len(stvUser.Data.UserByConnection.EmoteSets) > 0 {
						for _, emote := range stvUser.Data.UserByConnection.EmoteSets[0].Emotes {
							tempFile, err := os.CreateTemp("", "")
							if err != nil {
								log.Printf("Failed to create temp file %s for 7tv emote %s", err, emote.Data.Name)
								continue
							}
							e := Emote{
								Id:        emote.Data.ID,
								Source:    SevenTV,
								Name:      emote.Data.Name,
								Animated:  emote.Data.Animated,
								ChannelID: &channelId,
								OwnerID:   "",
								Owner:     nil,
								TempFile:  tempFile.Name(),
							}
							tempFile.Close()
							emoteObjs = append(emoteObjs, e)
						}
					}
					log.Println("Loaded " + strconv.Itoa(len(emoteObjs)) + " 7tv user emotes for " + channelId)
					if len(emoteObjs) > 0 {
						result := tx.Create(&emoteObjs)
						if result.Error != nil {
							return result.Error
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("Failed to add temp tabs: %s", err)
		}
	}(channelIds)

	err := hc.memDB.Transaction(func(tx *gorm.DB) error {
		var tempTabs []TempTab
		for _, channelId := range *channelIds {
			tempTabs = append(tempTabs, TempTab{
				Id:        channelId,
				ChatUsers: []ChatUser{},
			})
		}
		result := tx.Create(&tempTabs)
		if result.Error != nil {
			return result.Error
		}
		return nil
	})

	return err
}

func (hc *HasherinoController) GetEmotes() ([]*Emote, error) {
	emotes := []*Emote{}
	err := hc.memDB.Transaction(func(tx *gorm.DB) error {
		tab := &Tab{}
		result := hc.permDB.Take(&tab, "Selected = ?", true)
		if result.Error != nil {
			return result.Error
		}
		activeAccount := &Account{}
		result = hc.permDB.Take(&activeAccount, "Active = ?", true)
		if result.Error != nil {
			return result.Error
		}
		query := "(owner_id = ? OR owner_id IS ?) AND (channel_id = ? OR channel_id IS NULL)"
		result = tx.Where(query, activeAccount.Id, "", tab.Id).Find(&emotes)
		log.Println("Query found " + strconv.Itoa(len(emotes)) + " emotes for tab " + tab.Login)
		if result.Error != nil {
			return result.Error
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return emotes, nil
}

func (hc *HasherinoController) RemoveTab(id string) error {
	err := hc.permDB.Transaction(func(tx *gorm.DB) error {
		tab := &Tab{}
		result := tx.Take(&tab, "Id = ?", id)
		if result.Error != nil {
			return errors.New("Tab not found for id " + id)
		}

		result = tx.Delete(&tab)
		if result.Error != nil {
			return result.Error
		}

		result = hc.memDB.Delete(&TempTab{}, "Id = ?", id)
		if result.Error != nil {
			return result.Error
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

func (hc *HasherinoController) SetSelectedTab(login string) error {
	return hc.permDB.Transaction(func(tx *gorm.DB) error {
		selectedTabs := []*Tab{}
		result := tx.Find(&selectedTabs, "Selected = ?", true)
		if result.Error != nil {
			return result.Error
		}
		if len(selectedTabs) > 0 {
			for _, tab := range selectedTabs {
				tab.Selected = false
				result = tx.Save(&tab)
				if result.Error != nil {
					return result.Error
				}
			}
		}
		tab := &Tab{}
		result = hc.permDB.Take(&tab, "Login = ?", login)
		if result.Error != nil {
			return result.Error
		}
		tab.Selected = true
		result = tx.Save(&tab)
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
}

func (hc *HasherinoController) GetSelectedTab() (*Tab, error) {
	tab := &Tab{}
	result := hc.permDB.Take(&tab, "Selected = ?", true)
	return tab, result.Error
}

func (hc *HasherinoController) Listen() error {
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

	callbackWrapper := func(message string) {
		msg, err := ParseMessage(message)
		if err != nil {
			log.Printf("Failed to parse message: %s", err)
			return
		}

		callback, ok := hc.callbackMap[msg.Channel]
		if !ok {
			log.Printf("No callback for channel %s.", msg.Channel)
			return
		}
		callback(*msg)
	}

	for channel := range hc.callbackMap {
		_, joinedChannel := hc.chatWS.channels[channel]
		if !joinedChannel {
			hc.chatWS.Join(channel)
		}
	}

	go hc.chatWS.Listen(callbackWrapper)
	return nil
}

func (hc *HasherinoController) IsChannelJoined(channel string) bool {
	if hc.chatWS == nil || hc.chatWS.State == Disconnected {
		return false
	}

	_, joinedChannel := hc.chatWS.channels[channel]
	return joinedChannel
}

func (hc *HasherinoController) SendMessage(channel string, message string) error {
	err := hc.chatWS.Send(channel, message)
	return err
}

func (hc *HasherinoController) GetSettings() (*AppSettings, error) {
	appSettings := &AppSettings{}
	result := hc.permDB.Take(appSettings)
	return appSettings, result.Error
}

func (hc *HasherinoController) SetSettings(appSettings *AppSettings) error {
	return hc.permDB.Save(appSettings).Error
}

func GetDataFolder() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Local", "Hasherino"), nil
	case "linux":
		return filepath.Join(home, ".local", "share", "hasherino"), nil
	default:
		return home, nil
	}
}
