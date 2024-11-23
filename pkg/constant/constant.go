package constant

const (
	ROLE_ID_ADMIN         = 1
	ROLE_ID_KEPALA_DIVISI = 2
	ROLE_ID_STAF          = 3

	REDIS_REQUEST_IP_KEYS      = "reset-password:ip:%s"
	REDIS_REQUEST_MAX_ATTEMPTS = 5
	REDIS_REQUEST_IP_EXPIRE    = 240
)

var (
	BASE_URL string = ""
)
