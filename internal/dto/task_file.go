package dto

import "mime/multipart"

type TaskFileCreateRequest struct {
	TaskId *int `json:"task_id" form:"task_id" validate:"required"`
	File   []*multipart.FileHeader
}

type TaskFileFindByTaskIDRequest struct {
	TaskId int `param:"task_id" validate:"required"`
}

type TaskFileDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type TaskFileUpdateRequest struct {
	ID   int     `param:"id" validate:"required"`
	Name *string `json:"name" form:"name"`
}

type TaskFileFindByIDRequest struct {
	ID int `param:"id" validate:"required"`
}
