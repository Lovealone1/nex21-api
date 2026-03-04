package domain

import (
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

// ChartOfAccount represents the public.chart_of_accounts table
type ChartOfAccount struct {
	db.TenantBaseModel

	Code        string  `gorm:"type:text;not null;index:uq_tenant_coa_code,unique"`
	Name        string  `gorm:"type:text;not null"`
	Description *string `gorm:"type:text"`

	AccountType   string `gorm:"type:text;not null"` // 'asset', 'liability', 'equity', 'income', 'expense'
	NormalBalance string `gorm:"type:text;not null"` // 'debit', 'credit'

	ParentID *string `gorm:"type:uuid"`
	IsActive bool    `gorm:"not null;default:true"`
}

// LedgerJournal represents the public.ledger_journals table (Header)
type LedgerJournal struct {
	db.TenantBaseModel

	JournalNumber string  `gorm:"type:text;not null;index:uq_tenant_journal_number,unique"`
	Description   *string `gorm:"type:text"`

	Status      string     `gorm:"type:text;not null;default:'draft'"`
	JournalDate time.Time  `gorm:"type:date;not null"`
	PostedAt    *time.Time `gorm:"type:timestamptz"`
	PostedBy    *string    `gorm:"type:uuid"`

	SourceType *string `gorm:"type:text"`
	SourceID   *string `gorm:"type:uuid"`

	TotalDebit  float64 `gorm:"type:numeric(12,2);not null;default:0"`
	TotalCredit float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency    string  `gorm:"type:text;not null;default:'COP'"`

	Entries []LedgerEntry `gorm:"foreignKey:JournalID"`
}

// LedgerEntry represents the public.ledger_entries table (Lines)
type LedgerEntry struct {
	db.TenantBaseModel

	JournalID string `gorm:"type:uuid;not null"`
	AccountID string `gorm:"type:uuid;not null"`

	Description *string `gorm:"type:text"`
	Debit       float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Credit      float64 `gorm:"type:numeric(12,2);not null;default:0"`
	Currency    string  `gorm:"type:text;not null;default:'COP'"`

	ContactID  *string `gorm:"type:uuid"`
	LocationID *string `gorm:"type:uuid"`
	StaffID    *string `gorm:"type:uuid"`
}
