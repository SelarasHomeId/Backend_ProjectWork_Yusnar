package dto

import (
	"mime/multipart"
)

type TaskCreateRequest struct {
	BoardId *int    `json:"board_id" form:"board_id" validate:"required"`
	Title   *string `json:"title" form:"title" validate:"required"`
}

type TaskDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type TaskFindByBoardIDRequest struct {
	BoardID int `param:"board_id" validate:"required"`
}

type TaskUpdateRequest struct {
	ID           int     `param:"id" validate:"required"`
	BoardId      *int    `json:"board_id" form:"board_id"`
	Title        *string `json:"title" form:"title"`
	Description  *string `json:"description" form:"description"`
	AssignToUser []int   `json:"assign_to_user" form:"assign_to_user"`
	Label        *string `json:"label" form:"label"`
	IsCompleted  *bool   `json:"is_completed" form:"is_completed"`
	DueDate      *string `json:"due_date" form:"due_date"`
	Cover        []*multipart.FileHeader
}
