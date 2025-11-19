package foundation

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
)

func isGCPTokenValid(tokenString string) bool {
	if utils.IsEmptyStr(tokenString) {
		return false
	}

	if isGCPTokenExpired(tokenString) {
		return false
	}

	return true
}

// Función para comprobar si el token está caducado
func isGCPTokenExpired(tokenString string) bool {
	// Parsear el token sin verificar la firma, solo para obtener el payload
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &jwt.MapClaims{})

	if err != nil {
		return false
	}

	// Obtener las reclamaciones del token
	if claims, ok := token.Claims.(*jwt.MapClaims); ok {
		// Obtener el tiempo de expiración
		if exp, ok := (*claims)["exp"].(float64); ok {
			// Comprobar si el token ha expirado
			expirationTime := time.Unix(int64(exp), 0)
			return time.Now().After(expirationTime)
		}
	}
	return false
}

func getNewGCPAccessToken(tokenType string) (*oauth2.Token, error) {
	// Try environment variable first
	envPath := utils.GetEnv("ROOT_FOLDER")

	var serviceAccountFile string

	// If environment variable is not set, try to use current working directory
	if envPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current working directory: %v", err)
		}

		envPath = currentDir
	}

	switch tokenType {
	case "instances":
		serviceAccountFile = "cgp_instances_key.json"
	case "storage":
		serviceAccountFile = "cgp_storage_key.json"
	case "cloudbuild":
		serviceAccountFile = "cgp_cloudbuild_key.json"
	case "cloudscheduler":
		serviceAccountFile = "cgp_cloudscheduler_key.json"
	default:
		return nil, fmt.Errorf("GPCController.GetNewGCPAccessToken: invalid token type")
	}

	// Read the JSON file
	data, err := os.ReadFile(serviceAccountFile)
	if err != nil {
		return nil, fmt.Errorf("error reading service account JSON file: %v. File: %s", err, serviceAccountFile)
	}

	// Load credentials from JSON
	credentials, err := google.CredentialsFromJSON(context.Background(), data, iam.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("error loading credentials: %v", err)
	}

	// Get access token
	token, err := credentials.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error obtaining access token: %v", err)
	}

	return token, nil
}

func GetGCPToken(token string, tokenType string, user User) (*oauth2.Token, error) {

	if !user.IsStaff() && !user.IsSystem() {
		return nil, fmt.Errorf("GPCController.GetGCPToken: user is not staff")
	}

	if isGCPTokenValid(token) {
		return &oauth2.Token{AccessToken: token}, nil
	}
	// Obtener un nuevo token
	accessToken, err := getNewGCPAccessToken(tokenType)
	if err != nil {
		err = fmt.Errorf("Error al obtener un nuevo token: %v", err)
		log.Err(err)
		return nil, err
	}

	return accessToken, nil

}
