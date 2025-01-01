package dto

type TaskCommentCreateRequest struct {
	TaskId  *int    `json:"task_id" form:"task_id" validate:"required"`
	Comment *string `json:"comment" form:"comment" validate:"required"`
}

type TaskCommentFindByTaskIDRequest struct {
	TaskId int `param:"task_id" validate:"required"`
}

type TaskCommentDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type TaskCommentUpdateRequest struct {
	ID      int     `param:"id" validate:"required"`
	Comment *string `json:"comment" form:"comment"`
}

type TaskCommentFindByIDRequest struct {
	ID int `param:"id" validate:"required"`
}
