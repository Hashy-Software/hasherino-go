package hasherino

import (
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	Id     string
	Login  string
	Active bool
	// TODO: enable db encryption
	Token string
}
