package dto

type AccessCreateRequest struct {
	Module  *string `param:"module"`
	Phone   *string `param:"phone"`
	Message *string `param:"message"`
}

type AccessCreateResponse struct {
	Module  *string `param:"module"`
	Phone   *string `param:"phone"`
	Message *string `param:"message"`
}

type AccesstCountResponse struct {
	CountInstagram int `json:"count_instagram"`
	CountTiktok    int `json:"count_tiktok"`
	CountFacebook  int `json:"count_facebook"`
	CountWhatsapp  int `json:"count_whatsapp"`
	CountAffiliate int `json:"count_affiliate"`
	CountContact   int `json:"count_contact"`
}
