package main

import (
	"fmt"
	"log"

	"github.com/Lovealone1/nex21-api/internal/modules/catalog/domain"
	crmDomain "github.com/Lovealone1/nex21-api/internal/modules/crm/domain"
	financeDomain "github.com/Lovealone1/nex21-api/internal/modules/finance/domain"
	iamDomain "github.com/Lovealone1/nex21-api/internal/modules/iam/domain"
	inventoryDomain "github.com/Lovealone1/nex21-api/internal/modules/inventory/domain"
	locDomain "github.com/Lovealone1/nex21-api/internal/modules/locations/domain"
	payrollDomain "github.com/Lovealone1/nex21-api/internal/modules/payroll/domain"
	purchasingDomain "github.com/Lovealone1/nex21-api/internal/modules/purchasing/domain"
	salesDomain "github.com/Lovealone1/nex21-api/internal/modules/sales/domain"
	schedDomain "github.com/Lovealone1/nex21-api/internal/modules/scheduling/domain"
	tenantDomain "github.com/Lovealone1/nex21-api/internal/modules/tenant/domain"

	"github.com/Lovealone1/nex21-api/internal/platform/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration to get the real DB URL
	cfg := config.Load()

	// Initialize GORM with DryRun to just check schema mappings without altering DB
	db, err := gorm.Open(postgres.Open(cfg.DBUrl), &gorm.Config{
		DryRun: true,
	})

	if err != nil {
		log.Fatal("Failed to initialize GORM dry run:", err)
	}

	fmt.Println("Starting dry run validation for all GORM models to check tags/syntax...")

	err = db.AutoMigrate(
		// IAM
		&iamDomain.Profile{},

		// Tenant
		&tenantDomain.Tenant{},
		&tenantDomain.TenantDomain{},
		&tenantDomain.Membership{},

		// Locations & CRM
		&locDomain.Location{},
		&crmDomain.Contact{},

		// Catalog & Inventory
		&domain.CatalogItem{},
		&inventoryDomain.InventoryItem{},

		// Scheduling
		&schedDomain.Staff{},
		&schedDomain.WorkSchedule{},
		&schedDomain.Service{},
		&schedDomain.Appointment{},

		// Sales & Purchasing
		&salesDomain.SalesOrder{},
		&salesDomain.SalesOrderLine{},
		&purchasingDomain.PurchaseOrder{},
		&purchasingDomain.PurchaseOrderLine{},

		// Payroll
		&payrollDomain.StaffCompensation{},
		&payrollDomain.PayrollRun{},
		&payrollDomain.PayrollItem{},

		// Finance
		&financeDomain.Account{},
		&financeDomain.ChartOfAccount{},
		&financeDomain.LedgerJournal{},
		&financeDomain.LedgerEntry{},
		&financeDomain.ExpenseCategory{},
		&financeDomain.Expense{},
		&financeDomain.Transaction{},
	)

	if err != nil {
		log.Fatalf("Schema validation failed: %v\n", err)
	}

	fmt.Println("All models parsed successfully by GORM!")
}
