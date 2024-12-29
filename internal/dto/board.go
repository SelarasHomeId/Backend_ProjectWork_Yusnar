package dto

type BoardCreateRequest struct {
	WorkspaceId *int    `json:"workspace_id" form:"workspace_id" validate:"required"`
	Name        *string `json:"name" form:"name" validate:"required"`
}

type BoardDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type BoardUpdateRequest struct {
	ID          int     `param:"id" validate:"required"`
	WorkspaceId *int    `json:"workspace_id" form:"workspace_id"`
	Name        *string `json:"name" form:"name"`
	SortNumber  *int    `json:"sort_number" form:"sort_number"`
}

type BoardFindByWorkspaceIDRequest struct {
	WorkspaceID int `param:"workspace_id" validate:"required"`
}
