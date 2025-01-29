package dto

type TaskChecklistCreateRequest struct {
	TaskId int    `json:"task_id" form:"task_id"`
	Title  string `json:"title" form:"title"`
}

type TaskChecklistFindByTaskIDRequest struct {
	TaskId int `param:"task_id" validate:"required"`
}

type TaskChecklistUpdateRequest struct {
	ID    int     `param:"id" validate:"required"`
	Title *string `json:"title" form:"title"`
}

type TaskChecklistDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}
