package factory

import (
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/database"

	"gorm.io/gorm"
)

type Factory struct {
	Db *gorm.DB

	// repository
	Repository_initiated
}

type Repository_initiated struct {
	TestRepository         repository.Test
	ContactRepository      repository.Contact
	AffiliateRepository    repository.Affiliate
	AdminRepository        repository.Admin
	NotificationRepository repository.Notification
	AccessRepository       repository.Access
}

func NewFactory() *Factory {
	f := &Factory{}
	f.SetupDb()
	f.SetupRepository()
	return f
}

func (f *Factory) SetupDb() {
	db, err := database.Connection("MYSQL")
	if err != nil {
		panic("Failed setup db, connection is undefined")
	}
	f.Db = db
}

func (f *Factory) SetupRepository() {
	if f.Db == nil {
		panic("Failed setup repository, db is undefined")
	}

	f.TestRepository = repository.NewTest(f.Db)
	f.ContactRepository = repository.NewContact(f.Db)
	f.AffiliateRepository = repository.NewAffiliate(f.Db)
	f.AdminRepository = repository.NewAdmin(f.Db)
	f.NotificationRepository = repository.NewNotification(f.Db)
	f.AccessRepository = repository.NewAccess(f.Db)
}
