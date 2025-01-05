package factory

import (
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/database"
	"selarashomeid/pkg/gdrive"

	"github.com/go-redis/redis/v8"
	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Factory struct {
	Db *gorm.DB

	DbRedis *redis.Client

	GDrive GoogleDrive

	// repository
	Repository_initiated
}

type Repository_initiated struct {
	BannerRepository      repository.Banner
	UserRepository        repository.User
	DivisiRepository      repository.Divisi
	RoleRepository        repository.Role
	NotifikasiRepository  repository.Notifikasi
	WorkspaceRepository   repository.Workspace
	BoardRepository       repository.Board
	TaskRepository        repository.Task
	TaskFileRepository    repository.TaskFile
	TaskCommentRepository repository.TaskComment
	AccessRepository      repository.Access
	ContactRepository     repository.Contact
	AffiliateRepository   repository.Affiliate
}

type GoogleDrive struct {
	Service *drive.Service
	Folder  *drive.File
}

func NewFactory() *Factory {
	f := &Factory{}
	f.SetupDb()
	f.SetupDbRedis()
	f.SetupGoogleDrive()
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

func (f *Factory) SetupGoogleDrive() {
	service, folder, err := gdrive.InitGoogleDrive()
	if err != nil {
		panic("Failed setup gdrive, connection is undefined")
	}
	f.GDrive.Service = service
	f.GDrive.Folder = folder
}

func (f *Factory) SetupRepository() {
	if f.Db == nil {
		panic("Failed setup repository, db is undefined")
	}

	f.UserRepository = repository.NewUser(f.Db)
	f.DivisiRepository = repository.NewDivisi(f.Db)
	f.RoleRepository = repository.NewRole(f.Db)
	f.NotifikasiRepository = repository.NewNotifikasi(f.Db)
	f.BannerRepository = repository.NewBanner(f.Db)
	f.WorkspaceRepository = repository.NewWorkspace(f.Db)
	f.BoardRepository = repository.NewBoard(f.Db)
	f.TaskRepository = repository.NewTask(f.Db)
	f.TaskFileRepository = repository.NewTaskFile(f.Db)
	f.TaskCommentRepository = repository.NewTaskComment(f.Db)
	f.AccessRepository = repository.NewAccess(f.Db)
	f.ContactRepository = repository.NewContact(f.Db)
	f.AffiliateRepository = repository.NewAffiliate(f.Db)
}
