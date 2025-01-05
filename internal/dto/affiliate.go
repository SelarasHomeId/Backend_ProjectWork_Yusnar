package dto

type AffiliateCreateRequest struct {
	Name      *string `json:"name" form:"name"`
	Email     *string `json:"email" form:"email"`
	Phone     *string `json:"phone" form:"phone"`
	Instagram *string `json:"instagram" form:"instagram"`
	Tiktok    *string `json:"tiktok" form:"tiktok"`
}
