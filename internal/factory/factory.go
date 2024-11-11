package factory

import (
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/database"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Factory struct {
	Db *gorm.DB

	DbRedis *redis.Client

	// repository
	Repository_initiated
}

type Repository_initiated struct {
	TestRepository   repository.Test
	UserRepository   repository.User
	DivisiRepository repository.Divisi
	RoleRepository   repository.Role
}

func NewFactory() *Factory {
	f := &Factory{}
	f.SetupDb()
	f.SetupDbRedis()
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

func (f *Factory) SetupDbRedis() {
	dbRedis := database.InitRedis()
	f.DbRedis = dbRedis
}

func (f *Factory) SetupRepository() {
	if f.Db == nil {
		panic("Failed setup repository, db is undefined")
	}

	f.TestRepository = repository.NewTest(f.Db)
	f.UserRepository = repository.NewUser(f.Db)
	f.DivisiRepository = repository.NewDivisi(f.Db)
	f.RoleRepository = repository.NewRole(f.Db)
}
