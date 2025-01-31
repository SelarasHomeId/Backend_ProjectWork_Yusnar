package dto

type WorkspaceGetDataRequest struct {
	ID int `param:"id" validate:"required"`
}
