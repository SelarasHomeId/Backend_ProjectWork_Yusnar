package test

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/gomail"
	"selarashomeid/pkg/util/response"

	"gorm.io/gorm"
)

type Service interface {
	Test(*abstraction.Context) (*dto.TestResponse, error)
	TestGomail(*abstraction.Context, string) (*dto.TestResponse, error)
	TestDrive(*abstraction.Context, []*multipart.FileHeader) (*dto.TestResponse, error)
}

type service struct {
	Repository repository.Test
	Db         *gorm.DB
}

func NewService(f *factory.Factory) Service {
	repository := f.TestRepository
	db := f.Db
	return &service{
		repository,
		db,
	}
}

func (s *service) Test(ctx *abstraction.Context) (*dto.TestResponse, error) {
	result := dto.TestResponse{
		Message: "Success",
	}
	return &result, nil
}

func (s *service) TestGomail(ctx *abstraction.Context, recipient string) (*dto.TestResponse, error) {
	err := gomail.SendMail(recipient)
	if err != nil {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	result := dto.TestResponse{
		Message: "Success",
	}
	return &result, nil
}

func (s *service) TestDrive(ctx *abstraction.Context, files []*multipart.FileHeader) (*dto.TestResponse, error) {
	srvDrive, err := gdrive.InitGoogleDrive()
	if err != nil {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	newFolder, err := gdrive.FolderToDrive(srvDrive, "Example New Folder", "root")
	if err != nil {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	var uploadedFiles []string
	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		defer f.Close()

		newFile, err := gdrive.FileToDrive(srvDrive, file.Filename, "application/octet-stream", f, newFolder.Id)
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		uploadedFiles = append(uploadedFiles, newFile.Name)
	}

	result := dto.TestResponse{
		Message: fmt.Sprintf("Files '%v' uploaded successfully", uploadedFiles),
	}
	return &result, nil
}
