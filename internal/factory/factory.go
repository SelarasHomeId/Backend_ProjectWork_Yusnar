package factory

import (
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/database"
	"selarashomeid/pkg/gdrive"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Factory struct {
	Db *gorm.DB

	DbRedis *redis.Client

	GDrive GoogleDrive

	Repository_initiated
}

type Repository_initiated struct {
	BannerRepository        repository.Banner
	UserRepository          repository.User
	DivisiRepository        repository.Divisi
	RoleRepository          repository.Role
	NotifikasiRepository    repository.Notifikasi
	WorkspaceRepository     repository.Workspace
	BoardRepository         repository.Board
	TaskRepository          repository.Task
	TaskFileRepository      repository.TaskFile
	TaskCommentRepository   repository.TaskComment
	AccessRepository        repository.Access
	ContactRepository       repository.Contact
	AffiliateRepository     repository.Affiliate
	TaskLabelRepository     repository.TaskLabel
	TaskChecklistRepository repository.TaskChecklist
	ChecklistItemRepository repository.ChecklistItem
	ProjectRepository       repository.Project
}

type GoogleDrive struct {
	Service             *drive.Service
	FolderSelarasHomeId *drive.File
	FolderAttachment    *drive.File
	FolderProjectCover  *drive.File
	FolderTaskCover     *drive.File
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

	// sqlDB, err := db.DB()
	// if err != nil {
	// 	panic(err)
	// }
	// sqlDB.SetMaxIdleConns(5)
	// sqlDB.SetMaxOpenConns(20)
	// sqlDB.SetConnMaxLifetime(time.Hour)

	f.Db = db
}

func (f *Factory) SetupDbRedis() {
	dbRedis := database.InitRedis()
	f.DbRedis = dbRedis
}

func (f *Factory) SetupGoogleDrive() {
	service, err := gdrive.InitService()
	if err != nil {
		panic("Failed setup gdrive, connection is undefined")
	}
	folderSelarasHomeId, err := gdrive.InitFolder(service, constant.DRIVE_FOLDER, "root")
	if err != nil {
		logrus.Infof("Failed setup folder %s, cause: %s", constant.DRIVE_FOLDER, err.Error())
	}
	folderAttachment, err := gdrive.InitFolder(service, "attachment", folderSelarasHomeId.Id)
	if err != nil {
		logrus.Infof("Failed setup folder %s, cause: %s", "attachment", err.Error())
	}
	folderProjectCover, err := gdrive.InitFolder(service, "project_cover", folderSelarasHomeId.Id)
	if err != nil {
		logrus.Infof("Failed setup folder %s, cause: %s", "project_cover", err.Error())
	}
	folderTaskCover, err := gdrive.InitFolder(service, "task_cover", folderSelarasHomeId.Id)
	if err != nil {
		logrus.Infof("Failed setup folder %s, cause: %s", "task_cover", err.Error())
	}
	f.GDrive.Service = service
	f.GDrive.FolderSelarasHomeId = folderSelarasHomeId
	f.GDrive.FolderAttachment = folderAttachment
	f.GDrive.FolderProjectCover = folderProjectCover
	f.GDrive.FolderTaskCover = folderTaskCover
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
	f.TaskLabelRepository = repository.NewTaskLabel(f.Db)
	f.TaskChecklistRepository = repository.NewTaskChecklist(f.Db)
	f.ChecklistItemRepository = repository.NewChecklistItem(f.Db)
	f.ProjectRepository = repository.NewProject(f.Db)
}
