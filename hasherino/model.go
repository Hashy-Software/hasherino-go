package hasherino

type Account struct {
	Id          string `gorm:"primaryKey"`
	Login       string
	DisplayName string
	Active      bool
	Token       string // TODO: enable db encryption to hide token
}
