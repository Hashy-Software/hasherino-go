package hasherino

import "gorm.io/gorm"

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
}
