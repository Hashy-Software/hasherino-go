package hasherino

import (
	"errors"

	"gorm.io/gorm"
)

// --- permDB models ---
type Account struct {
	Id          string `gorm:"primaryKey"`
	Login       string
	DisplayName string
	Active      bool
	Token       string // TODO: enable db encryption to hide token
}

type Tab struct {
	Id          string `gorm:"primaryKey"`
	Login       string
	DisplayName string
	Selected    bool
}

// Single row table for global settings
type AppSettings struct {
	gorm.Model
	ChatMessageLimit int // Maximum amount of messages in a single chat
	ChatHistory      bool
}

// --- tempDB models ---
type EmoteSourceEnum int64

const (
	Twitch EmoteSourceEnum = iota
	SevenTV
)

// Handles tab data that is not persisted
type TempTab struct {
	Id        string     `gorm:"primaryKey"`
	ChatUsers []ChatUser `gorm:"many2many:chat_user_temp_tab;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
}

type ChatUser struct {
	Id          string `gorm:"primaryKey"`
	Login       string
	DisplayName string
	TempTabs    []TempTab `gorm:"many2many:chat_user_temp_tab;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
	Emotes      []Emote   `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
}

type ChatUserTempTab struct {
	ChatUserId string `gorm:"primaryKey"`
	TempTabId  string `gorm:"primaryKey"`
}

type Emote struct {
	Id     string          `gorm:"primaryKey"`
	Source EmoteSourceEnum `gorm:"primaryKey"`
	Name   string
	// if a channel is set, only renders in that channel
	ChannelID *string
	// Foreign key field
	OwnerID string `gorm:"index"`
	// if an owner is set, only renders when the message sender is the owner
	Owner *ChatUser `gorm:"foreignKey:OwnerID;references:Id"`
}

// When a TempTab gets deleted, delete all orphaned ChatUsers
func (c *ChatUserTempTab) AfterDelete(tx *gorm.DB) (err error) {
	var count int64
	if err := tx.Model(&ChatUserTempTab{}).Where("ChatUserId = ?", c.ChatUserId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := tx.Delete(&ChatUser{Id: c.ChatUserId}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (e *Emote) GetUrl() (string, error) {
	switch e.Source {
	case Twitch:
		return "https://static-cdn.jtvnw.net/emoticons/v2/" + e.Id + "/default/dark/2.0", nil
	case SevenTV:
		return "https://cdn.7tv.app/emote/" + e.Id + "/2x.webp", nil
	default:
		return "", errors.New("Unknown emote source")
	}
}
