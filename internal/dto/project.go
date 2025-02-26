package dto

import "mime/multipart"

type ProjectCreateRequest struct {
	Name     string  `json:"name" form:"name" validate:"required"`
	Location *string `json:"location" form:"location"`
	Cover    []*multipart.FileHeader
}

type ProjectUpdateRequest struct {
	ID          int     `param:"id" validate:"required"`
	Name        *string `json:"name" form:"name"`
	Location    *string `json:"location" form:"location"`
	Cover       []*multipart.FileHeader
	DeleteCover *bool `json:"delete_cover" form:"delete_cover"`
}

type ProjectDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type ProjectFindByIDRequest struct {
	ID int `param:"id" validate:"required"`
}
