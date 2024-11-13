package dto

type RoleUpdateRequest struct {
	ID   int     `param:"id" validate:"required"`
	Name *string `json:"name" form:"name"`
}
