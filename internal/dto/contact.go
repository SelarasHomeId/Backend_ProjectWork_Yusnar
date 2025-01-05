package dto

type ContactCreateRequest struct {
	Name    *string `json:"name" form:"name"`
	Email   *string `json:"email" form:"email"`
	Phone   *string `json:"phone" form:"phone"`
	Message *string `json:"message" form:"message"`
}
