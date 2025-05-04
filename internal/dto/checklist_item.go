package dto

type ChecklistItemCreateRequest struct {
	TaskChecklistId int    `json:"task_checklist_id" form:"task_checklist_id"`
	Title           string `json:"title" form:"title"`
}

type ChecklistItemFindByChecklistIDRequest struct {
	TaskChecklistId int `param:"task_checklist_id" validate:"required"`
}

type ChecklistItemUpdateRequest struct {
	ID              int     `param:"id" validate:"required"`
	Title           *string `json:"title" form:"title"`
	TaskChecklistId *int    `json:"task_checklist_id" form:"task_checklist_id"`
	AssignToUser    []int   `json:"assign_to_user" form:"assign_to_user"`
	IsCompleted     *bool   `json:"is_completed" form:"is_completed"`
	DueDate         *string `json:"due_date" form:"due_date"`
	SortNumber      *int    `json:"sort_number" form:"sort_number"`
}

type ChecklistItemDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type ChecklistItemConvertToTaskRequest struct {
	ID int `param:"id" validate:"required"`
}
