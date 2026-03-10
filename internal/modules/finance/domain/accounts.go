package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

type AccountType string

const (
	AccountTypeCash         AccountType = "cash"
	AccountTypeBank         AccountType = "bank"
	AccountTypeWallet       AccountType = "wallet"
	AccountTypeCardTerminal AccountType = "card_terminal"
	AccountTypeGateway      AccountType = "gateway"
	AccountTypeOther        AccountType = "other"
)

// Account represents the public.accounts table.
type Account struct {
	db.BaseModel
	TenantID string `gorm:"type:uuid;not null;index:accounts_tenant_id_idx;index:accounts_tenant_type_idx;index:accounts_tenant_name_idx"`

	Name string  `gorm:"type:text;not null"`
	Code *string `gorm:"type:text"`

	AccountType AccountType `gorm:"type:text;not null;default:cash"`
	Currency    string      `gorm:"type:text;not null;default:COP"`

	IsActive  bool `gorm:"type:boolean;not null;default:true"`
	IsDefault bool `gorm:"type:boolean;not null;default:false"`

	Provider *string `gorm:"type:text"`
	Notes    *string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Account) TableName() string {
	return "accounts"
}
