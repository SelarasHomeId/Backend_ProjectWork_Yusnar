package dto

type ProjectCreateRequest struct {
	Name     string  `json:"name" form:"name" validate:"required"`
	Location *string `json:"location" form:"location"`
}

type ProjectUpdateRequest struct {
	ID       int     `param:"id" validate:"required"`
	Name     *string `json:"name" form:"name"`
	Location *string `json:"location" form:"location"`
}

type ProjectDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}
