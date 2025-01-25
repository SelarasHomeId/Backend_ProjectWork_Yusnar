package dto

type TaskLabelCreateRequest struct {
	Title *string `json:"title" form:"title"`
	Color string  `json:"color" form:"color"`
}

type TaskLabelFindByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type TaskLabelUpdateRequest struct {
	ID    int     `param:"id" validate:"required"`
	Title *string `json:"title" form:"title"`
	Color *string `json:"color" form:"color"`
}

type TaskLabelDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}
