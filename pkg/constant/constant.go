package constant

const (
	APP      = "selarashomeid"
	PORT     = "3000"
	VERSION  = "1.0.0"
	BASE_URL = "https://oryx-credible-buzzard.ngrok-free.app"

	ROLE_ID_ADMIN         = 1
	ROLE_ID_KEPALA_DIVISI = 2
	ROLE_ID_STAF          = 3

	REDIS_REQUEST_IP_KEYS      = "reset-password:ip:%s"
	REDIS_REQUEST_MAX_ATTEMPTS = 5
	REDIS_REQUEST_IP_EXPIRE    = 240
)
