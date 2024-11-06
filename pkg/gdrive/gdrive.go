package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"selarashomeid/internal/config"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func FileToDrive(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{
		MimeType: mimeType,
		Name:     name,
		Parents:  []string{parentId},
	}
	file, err := service.Files.Create(f).Media(content).Do()

	if err != nil {
		logrus.Println("Could not create file: " + err.Error())
		return nil, err
	}

	return file, nil
}

func FolderToDrive(service *drive.Service, name string, parentId string) (*drive.File, error) {
	d := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentId},
	}

	file, err := service.Files.Create(d).Do()

	if err != nil {
		logrus.Println("Could not create dir: " + err.Error())
		return nil, err
	}

	return file, nil
}

func InitGoogleDrive() (*drive.Service, error) {
	credentialsJson := config.Get().Drive.CredentialsDrive

	config, err := google.ConfigFromJSON([]byte(credentialsJson), drive.DriveScope)

	if err != nil {
		return nil, err
	}

	client := getClient(config)

	service, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		logrus.Printf("Cannot create the Google Drive service: %v\n", err)
		return nil, err
	}

	logrus.Info("Drive ready!")
	return service, err
}

func getClient(config *oauth2.Config) *http.Client {
	tok, err := tokenFromEnv()
	if err != nil {
		tok = getTokenFromWeb(config)
		saveTokenToEnv(tok)
		logrus.Info("Regenerate token!")
	}
	logrus.Info("Client found!")
	return config.Client(context.Background(), tok)
}

func tokenFromEnv() (*oauth2.Token, error) {
	tokenJSON := os.Getenv("TOKEN_DRIVE")
	if tokenJSON == "" {
		return nil, fmt.Errorf("TOKEN_DRIVE environment variable is not set")
	}
	tok := &oauth2.Token{}
	err := json.Unmarshal([]byte(tokenJSON), tok)
	return tok, err
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	refreshToken := os.Getenv("REFRESH_DRIVE")
	if refreshToken == "" {
		logrus.Printf("REFRESH_TOKEN environment variable is not set")
		return nil
	}

	tok := &oauth2.Token{RefreshToken: refreshToken}
	tokSource := config.TokenSource(context.Background(), tok)

	newToken, err := tokSource.Token()
	if err != nil {
		logrus.Printf("Unable to retrieve token from web: %v", err)
		return nil
	}

	return newToken
}

func saveTokenToEnv(token *oauth2.Token) {
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		logrus.Printf("Failed to marshal token: %v", err)
		return
	}
	os.Setenv("TOKEN_DRIVE", string(tokenJSON))
}
