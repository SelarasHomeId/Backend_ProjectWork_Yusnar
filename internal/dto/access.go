package dto

type AccessCreateRequest struct {
	Module *string `json:"module" query:"module"`
	Option *string `json:"option" query:"option"`
}

type AccessGetCountResponse struct {
	CountInstagram int `json:"count_instagram"`
	CountTiktok    int `json:"count_tiktok"`
	CountFacebook  int `json:"count_facebook"`
	CountWhatsapp  int `json:"count_whatsapp"`
}
