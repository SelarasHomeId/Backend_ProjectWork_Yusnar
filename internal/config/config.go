package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Configuration struct {
	DB      DB
	Logging Logging
	JWT     JWT
	Gomail  Gomail
	Drive   Drive
}

type DB struct {
	DbHost string
	DbUser string
	DbPass string
	DbPort string
	DbName string
}

type Logging struct {
	GormLevel   string
	LogrusLevel string
}

type JWT struct {
	SecretKey string
}

type Gomail struct {
	SmtpHost     string
	SmtpPort     string
	SenderName   string
	AuthEmail    string
	AuthPassword string
}

type Drive struct {
	CredentialsDrive  string
	RefreshTokenDrive string
}

var lock = &sync.Mutex{}
var defaultConfig Configuration

func Get() *Configuration {
	lock.Lock()
	defer lock.Unlock()
	return &defaultConfig
}

func Init() *Configuration {

	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("Production")
	} else {
		fmt.Println("Development")
	}

	defaultConfig.DB.DbHost = os.Getenv("DB_HOST")
	defaultConfig.DB.DbUser = os.Getenv("DB_USER")
	defaultConfig.DB.DbPass = os.Getenv("DB_PASS")
	defaultConfig.DB.DbPort = os.Getenv("DB_PORT")
	defaultConfig.DB.DbName = os.Getenv("DB_NAME")
	defaultConfig.Logging.GormLevel = os.Getenv("GORM_LEVEL")
	defaultConfig.Logging.LogrusLevel = os.Getenv("LOGRUS_LEVEL")
	defaultConfig.JWT.SecretKey = os.Getenv("SECRET_KEY")

	// on development
	defaultConfig.Gomail.SmtpHost = os.Getenv("SMTP_HOST")
	defaultConfig.Gomail.SmtpPort = os.Getenv("SMTP_PORT")
	defaultConfig.Gomail.SenderName = os.Getenv("SENDER_NAME")
	defaultConfig.Gomail.AuthEmail = os.Getenv("AUTH_EMAIL")
	defaultConfig.Gomail.AuthPassword = os.Getenv("AUTH_PASSWORD")
	defaultConfig.Drive.CredentialsDrive = os.Getenv("CREDENTIALS_DRIVE")
	defaultConfig.Drive.RefreshTokenDrive = os.Getenv("REFRESH_DRIVE")

	return &defaultConfig
}
