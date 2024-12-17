package dto

import "mime/multipart"

type BannerFindByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type BannerCreateRequest struct {
	Files []*multipart.FileHeader
}

type BannerUpdateRequest struct {
	ID       int     `param:"id" validate:"required"`
	FileName *string `json:"file_name" form:"file_name"`
	Files    []*multipart.FileHeader
}

type BannerDeleteByIDRequest struct {
	ID int `param:"id" validate:"required"`
}

type BannerUpdatePopupRequest struct {
	ID       int     `param:"id" validate:"required"`
	FileName *string `json:"file_name" form:"file_name"`
	Files    []*multipart.FileHeader
}

type BannerSetPopupRequest struct {
	Set string `param:"set" validate:"required"`
}
