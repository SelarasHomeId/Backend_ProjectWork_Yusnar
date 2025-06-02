package constant

const (
	DRIVE_FOLDER = "SelarasHomeId_App"

	ROLE_ID_ADMIN = 1
	ROLE_ID_STAFF = 2

	USER_ID_SYSTEM = 1

	BLANK_TASK_ID = 1

	REDIS_REQUEST_IP_KEYS      = "reset-password:ip:%s"
	REDIS_REQUEST_MAX_ATTEMPTS = 5
	REDIS_REQUEST_IP_EXPIRE    = 240
	REDIS_KEY_USER_LOGIN       = "login_token_user_"
	REDIS_KEY_AUTO_LOGOUT      = "user_auto_logout"
	REDIS_KEY_REFRESH_TOKEN    = "refresh-token:%s"
	REDIS_MAX_REFRESH_TOKEN    = 7
	REDIS_EXP_REFRESH_TOKEN    = 1440

	LINK_INSTAGRAM = "https://www.instagram.com/selarashome_id/"
	LINK_TIKTOK    = "https://www.tiktok.com/@selarashomeid"
	LINK_FACEBOOK  = "https://www.facebook.com/profile.php?id=100079489771455&locale=id_ID"
	LINK_WHATSAPP  = "https://wa.me/"

	PATH_FILE_SAVED    = "../file_saved"
	PATH_ASSETS_IMAGES = "assets/images"
	PATH_SHARE         = "/var/www/html/selarashomeid/share"
)

var (
	BASE_URL string = ""
)
