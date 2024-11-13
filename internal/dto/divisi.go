package dto

type DivisiCreateRequest struct {
	Name *string `json:"name" form:"name" validate:"required"`
}

type DivisiUpdateRequest struct {
	ID   int     `param:"id" validate:"required"`
	Name *string `json:"name" form:"name"`
}

type DivisiDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}
