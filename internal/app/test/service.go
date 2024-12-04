package test

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/gomail"
	"selarashomeid/pkg/util/response"

	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Test(*abstraction.Context) (*dto.TestResponse, error)
	TestGomail(*abstraction.Context, string) (*dto.TestResponse, error)
	TestDriveCreate(*abstraction.Context, []*multipart.FileHeader) (*dto.TestResponse, error)
	TestDriveGetById(*abstraction.Context, string) (map[string]interface{}, error)
}

type service struct {
	Repository repository.Test
	Db         *gorm.DB
	sDrive     *drive.Service
	fDrive     *drive.File
}

func NewService(f *factory.Factory) Service {
	repository := f.TestRepository
	db := f.Db
	sDrive := f.GDrive.Service
	fDrive := f.GDrive.Folder
	return &service{
		repository,
		db,
		sDrive,
		fDrive,
	}
}

func (s *service) Test(ctx *abstraction.Context) (*dto.TestResponse, error) {
	result := dto.TestResponse{
		Message: "Success",
	}
	return &result, nil
}

func (s *service) TestGomail(ctx *abstraction.Context, recipient string) (*dto.TestResponse, error) {
	err := gomail.SendMail(recipient, "Test Email", "Hello World!")
	if err != nil {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	result := dto.TestResponse{
		Message: "Success",
	}
	return &result, nil
}

func (s *service) TestDriveCreate(ctx *abstraction.Context, files []*multipart.FileHeader) (*dto.TestResponse, error) {
	var uploadedFiles []string
	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		defer f.Close()

		newFile, err := gdrive.CreateFile(s.sDrive, file.Filename, "application/octet-stream", f, s.fDrive.Id)
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		s.Repository.Create(ctx, &model.TestEntityModel{
			Context: ctx,
			TestEntity: model.TestEntity{
				FileId:   newFile.Id,
				FileName: newFile.Name,
				IsDelete: false,
			},
		})

		uploadedFiles = append(uploadedFiles, newFile.Name)
	}

	result := dto.TestResponse{
		Message: fmt.Sprintf("Files '%v' uploaded successfully", uploadedFiles),
	}
	return &result, nil
}

func (s *service) TestDriveGetById(ctx *abstraction.Context, id string) (map[string]interface{}, error) {
	file, err := gdrive.GetFile(s.sDrive, id)
	if err != nil {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
	}

	return map[string]interface{}{
		"data": file,
	}, nil
}
