package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/weitecit/pkg/foundation"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
)

type mailData struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// getUserRoleFromMongoDB consulta MongoDB para obtener el rol real del usuario
func getUserRoleFromMongoDB(userID interface{}) string {
	if userID == nil {
		return ""
	}

	mongoRepo := foundation.MongoRepository{
		ConnectionString: utils.GetEnv("MONGO_REPO"),
		DataBase:         "main",
	}

	db, err := mongoRepo.GetDB()
	if err != nil || db == nil {
		return ""
	}

	collection := db.Collection("users")
	var result bson.M
	err = collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&result)
	if err != nil {
		return ""
	}

	rolePermData, ok := result["rolepermission"].(bson.M)
	if !ok {
		return ""
	}

	role, ok := rolePermData["role"].(string)
	if !ok {
		return ""
	}

	return role
}

// newBaseRequestWithModel allows tests to override foundation.NewBaseRequestWithModel
var newBaseRequestWithModel = func(model *foundation.User, user foundation.User) (*foundation.BaseRequest, error) {
	return foundation.NewBaseRequestWithModel(model, user)
}

// cloneHttpClient allows tests to override how HTTP client is created
var cloneHttpClient = func() *http.Client {
	// Configurar cliente HTTP con timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return client
}
var cloneUserModel = func() *foundation.User {
	return &foundation.User{}
}

func FillRequestFromToken(request *ServiceRequest) (*ServiceRequest, error) {

	if request.Token == "" {
		return request, errors.New("system.GetServiceRequestFromToken: token is empty")
	}

	claims, err := GetClaimsFromToken(request.Token)
	if err != nil {
		return request, errors.New("SystemService.GetServiceRequestFromToken: " + err.Error())
	}

	languageStr := utils.GetValueToStr(claims, "Language")
	userLanguageStr := utils.GetValueToStr(claims, "UserLanguage")
	if userLanguageStr == "" {
		userLanguageStr = languageStr
	}
	if languageStr == "" {
		languageStr = userLanguageStr
	}
	language, _ := foundation.NewLanguage(languageStr)

	request.RepoID = utils.GetValueToStr(claims, "DomainID")
	request.Language = language
	request.SpaceID = utils.GetValueToStr(claims, "SpaceID")
	request.Labels = utils.GetValueToArrayStr(claims, "Labels")

	user := &foundation.User{}
	err = user.GetFromMap(claims)
	request.User = *user

	return request, err
}

func getExpirationHours() int {
	expirationHours := utils.GetEnv("TOKEN_EXPIRATION_HOURS")
	if expirationHours == "" {
		return 24
	}
	return utils.StrToInt(expirationHours)
}

func CreateWebToken(user foundation.User) (string, error) {

	if !user.IsValid() {
		return "", errors.New("SystemService.CreateWebToken: user is not valid")
	}

	expiration := time.Now().Add(time.Hour * time.Duration(getExpirationHours())).Unix()

	// request.RepoModel = user
	// fRequest, err := NewFoundationBaseRequestWithRepository(request)
	// if err != nil {
	// 	return "", err
	// }

	// err = user.OpenSession(fRequest)
	// if err != nil {
	// 	return "", err
	// }

	type UserRole struct {
		PermissionID   string
		PermissionType float64
		Role           string
	}

	userRoles := []UserRole{}

	// Para usuarios client, consultar MongoDB para obtener el rol real
	if user.HasLabel(foundation.LabelClient) && user.RepoID != "" {
		// Consultar MongoDB para obtener el rol real del usuario
		roleFromDB := getUserRoleFromMongoDB(user.ID)
		if roleFromDB != "" {
			userRole := UserRole{
				PermissionID:   user.RepoID,
				PermissionType: float64(foundation.PermissionTypeFull),
				Role:           roleFromDB,
			}
			userRoles = append(userRoles, userRole)
		}
	} else if len(user.Roles) > 0 {
		// Para usuarios no-client, usar user.Roles si tiene elementos
		for _, permission := range user.Roles {
			userRole := UserRole{
				PermissionID:   permission.PermissionID,
				PermissionType: float64(permission.PermissionType),
				Role:           string(permission.Role),
			}
			userRoles = append(userRoles, userRole)
		}
	} else if user.RolePermission.PermissionID != "" {
		// Si RolePermission tiene datos, usarlo
		userRole := UserRole{
			PermissionID:   user.RolePermission.PermissionID,
			PermissionType: float64(user.RolePermission.PermissionType),
			Role:           string(user.RolePermission.Role),
		}
		userRoles = append(userRoles, userRole)
	}

	webToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"UserID":       user.ID,
		"ContactID":    user.ContactID,
		"DomainID":     user.RepoID,
		"Username":     user.Username,
		"UserLanguage": user.Language,
		"Language":     user.Language,
		"Roles":        userRoles,
		"UserLabels":   user.Labels,
		"Products":     user.Licenses,
		"Connection":   user.Connection,
		"SpaceID":      user.SpaceID,
		"exp":          expiration,
		"Nick":         user.Nick,
	})

	signedToken, err := webToken.SignedString([]byte(utils.GetEnv("SECRET_KEY")))
	if err != nil {
		return signedToken, err
	}

	return signedToken, nil
}
func CreateRefreshToken(userID string) (string, error) {

	if userID == " " {
		return "", errors.New("SystemService.CreateRefreshToken: userID not provided")
	}

	expiration := time.Now().Add(30 * time.Duration(getExpirationHours()) * time.Hour).Unix()

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"UserID": userID,
		"exp":    expiration,
	})

	signedRefreshToken, err := refreshToken.SignedString([]byte(utils.GetEnv("SECRET_KEY")))
	if err != nil {
		return signedRefreshToken, err
	}

	return signedRefreshToken, nil
}

func UpdateToken(token string) (string, error) {

	claims, err := DecodeToken(token)
	if err != nil && err.Error() != "error al decodificar token: Token is expired" {
		return "", err
	}

	expiration := time.Now().Add(time.Hour * time.Duration(getExpirationHours())).Unix()
	claims["exp"] = expiration

	updatedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := updatedToken.SignedString([]byte(utils.GetEnv("SECRET_KEY")))
	if err != nil {
		return "", err
	}

	return signedToken, nil

}

// TODO: que alguien estudie interfaces por favor
func SendEmail(to string, subject string, body string, channel log.HookChannel) error {

	mailData := mailData{
		From:    "it@weitec.es",
		To:      to,
		Subject: subject,
		Body:    body,
	}

	if channel == log.HookChannelNone {
		channel = log.HookChannelLog
	}

	err := sendMicrosoftGraphEmail(mailData)
	if err != nil {
		log.ToDiscord(channel, "❌ Error en SendEmail (Envío): "+err.Error())
		return fmt.Errorf("error al enviar el correo electrónico a %s sobre %s: %w", to, subject, err)
	}

	// Éxito: notificar a Discord
	successMsg := fmt.Sprintf("✅ Correo a %s sobre %s enviado exitosamente", to, subject)
	fmt.Println(successMsg)
	log.ToDiscord(channel, successMsg)
	return nil
}

// SendEmailRecovery envía un correo electrónico de recuperación de contraseña al destinatario especificado.
// Utiliza Microsoft Graph API para enviar el correo electrónico y notifica a Discord sobre el estado de la operación.
// Devuelve un error genérico si ocurre algún problema durante el proceso.
func SendEmailRecovery(to string, recoveryToken string) error {
	landingURI := utils.GetEnv("LANDING_URI")
	if landingURI == "" {
		errMsg := "Falta variable de entorno requerida: LANDING_URI"
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error de configuración del servidor")
	}

	resetURL := fmt.Sprintf("%s/reset-password/%s", landingURI, recoveryToken)
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Recuperación de contraseña</h2>
			<p>Has solicitado restablecer tu contraseña.</p>
			<p>Haz clic en el siguiente enlace para crear una nueva contraseña:</p>
			<p><a href="%s">Restablecer contraseña</a></p>
			<p>Si no has solicitado este cambio, puedes ignorar este mensaje.</p>
			<p>El enlace expirará en 1 hora.</p>
			<br>
			<p>Saludos,</p>
			<p>Equipo de Weitec</p>
		</body>
		</html>
	`, resetURL)

	log.ToDiscord(log.HookChannelLog, fmt.Sprintf("📤 Enviando correo de recuperación a: %s", to))

	mailData := mailData{
		From:    "it@weitec.es",
		To:      to,
		Subject: "Recuperación de contraseña - Weitec",
		Body:    body,
	}

	err := sendMicrosoftGraphEmail(mailData)
	if err != nil {
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail (Envío): "+err.Error())
		return fmt.Errorf("error al enviar el correo electrónico: %w", err)
	}

	// Éxito: notificar a Discord
	successMsg := fmt.Sprintf("✅ Correo de recuperación enviado exitosamente a: %s", to)
	fmt.Println(successMsg)
	log.ToDiscord(log.HookChannelLog, successMsg)
	return nil
}

func SendEmailMobile(to, recoveryCode string) error {

	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Código de recuperación</h2>
			<p>Has solicitado restablecer tu contraseña.</p>
			<p>Tu código de recuperación es: <strong>%s</strong></p>
			<p>El código expirará en 1 hora.</p>
			<br>
			<p>Saludos,</p>
			<p>Equipo de Weitec</p>
		</body>
		</html>
	`, recoveryCode)

	mailData := mailData{
		From:    "it@weitec.es",
		To:      to,
		Subject: "Código de recuperación - Weitec",
		Body:    body,
	}
	err := sendMicrosoftGraphEmail(mailData)
	if err != nil {
		return fmt.Errorf("error enviando email: %w", err)
	}

	fmt.Printf("Email de recuperación enviado exitosamente a: %s\n", to)
	return nil
}

// Estructura para la respuesta del token de Graph API
type graphTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// getMicrosoftGraphToken obtiene el token de acceso para Microsoft Graph API
func getMicrosoftGraphToken() (string, error) {
	clientID := utils.GetEnv("OAUTH_CLIENT_ID")
	clientSecret := utils.GetEnv("OAUTH_CLIENT_SECRET")
	tenantID := utils.GetEnv("OAUTH_TENANT_ID")
	microsoftClient := utils.GetEnv("MICROSOFT_CLIENT")

	if clientID == "" || clientSecret == "" || tenantID == "" || microsoftClient == "" {
		return "", fmt.Errorf("faltan variables de entorno requeridas para Microsoft Graph")
	}

	tokenURL := fmt.Sprintf(microsoftClient, tenantID)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("grant_type", "client_credentials")

	tokenReq, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creando request de token: %v", err)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := cloneHttpClient()
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return "", fmt.Errorf("error obteniendo token: %v", err)
	}
	defer tokenResp.Body.Close()

	bodyBytes, _ := io.ReadAll(tokenResp.Body)

	var tokenData graphTokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenData); err != nil {
		return "", fmt.Errorf("error decodificando respuesta de token: %v. Body: %s", err, string(bodyBytes))
	}

	if tokenData.Error != "" {
		errMsg := fmt.Sprintf("Error en la respuesta de autenticación: %s - %s", tokenData.Error, tokenData.Description)
		// Comprobar errores específicos si es necesario, similar a la implementación original
		if strings.Contains(strings.ToLower(tokenData.Description), "expir") || tokenData.Error == "invalid_grant" {
			return "", fmt.Errorf("el token de autenticación ha expirado")
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	if tokenData.AccessToken == "" {
		return "", fmt.Errorf("error al obtener token de acceso. Respuesta: %s", string(bodyBytes))
	}

	return tokenData.AccessToken, nil
}

// sendMicrosoftGraphEmail hace la llamada a Graph API para enviar el email
func sendMicrosoftGraphEmail(mailData mailData) error {

	if mailData.From == "" {
		mailData.From = "it@weitec.es"
	}

	graphEndpoint := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", mailData.From)

	emailData := map[string]interface{}{
		"message": map[string]interface{}{
			"subject":    mailData.Subject,
			"importance": "high",
			"body": map[string]interface{}{
				"contentType": "HTML",
				"content":     mailData.Body,
			},
			"toRecipients": []map[string]interface{}{
				{
					"emailAddress": map[string]string{
						"address": mailData.To,
					},
				},
			},
		},
		"saveToSentItems": true,
	}

	jsonData, err := json.Marshal(emailData)
	if err != nil {
		return fmt.Errorf("error creando JSON para el correo: %v", err)
	}

	mailReq, err := http.NewRequest("POST", graphEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creando solicitud de envío: %v", err)
	}

	accessToken, err := getMicrosoftGraphToken()
	if err != nil {
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail (Token): "+err.Error())
		return fmt.Errorf("error de configuración del servidor: %w", err) // Mensaje genérico para el frontend
	}

	mailReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	mailReq.Header.Set("Content-Type", "application/json")

	client := cloneHttpClient()
	resp, err := client.Do(mailReq)
	if err != nil {
		return fmt.Errorf("error en la petición de envío: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error del servidor Graph API (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func GenerateShortCode(length int) string {
	charset := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := make([]byte, length)
	rand.Seed(time.Now().UnixNano())
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func GenerateRecoveryToken(email string) (string, error) {
	expirationHours := "1" // El token expira en 1 hora
	expiration := time.Now().Add(time.Hour * time.Duration(utils.StrToInt(expirationHours))).Unix()

	webToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   expiration,
		"type":  "password_recovery",
	})

	jwtSecret := utils.GetEnv("SECRET_KEY")
	if jwtSecret == "" {
		jwtSecret = "tu_secreto_jwt" // Usar el mismo valor por defecto que en router.go
	}

	signedToken, err := webToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// ValidateRecoveryToken valida un token de recuperación de contraseña.
// Verifica que el token sea válido, no haya expirado y esté firmado correctamente.
// Devuelve true si el token es válido, o false y un error en caso contrario.
func ValidateRecoveryToken(tokenString string) (bool, error) {
	fmt.Printf("Validando token: %s\n", tokenString)
	jwtSecret := utils.GetEnv("SECRET_KEY")
	if jwtSecret == "" {
		return false, errors.New("SECRET_KEY no configurado")
	}
	fmt.Printf("SECRET_KEY: %s\n", jwtSecret)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		fmt.Printf("Error al parsear token: %v\n", err)
		return false, nil // Token inválido
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		fmt.Printf("Claims del token: %+v\n", claims)

		// Verificar que el token es para recuperación de contraseña
		if purpose, ok := claims["type"].(string); !ok || purpose != "password_recovery" {
			fmt.Printf("Tipo de token incorrecto. Esperado: password_recovery, Recibido: %v\n", purpose)
			return false, nil
		}

		// Verificar que el token no ha expirado
		if exp, ok := claims["exp"].(float64); !ok || int64(exp) < time.Now().Unix() {
			fmt.Printf("Token expirado. Exp: %v, Ahora: %v\n", int64(exp), time.Now().Unix())
			return false, nil
		}

		return true, nil
	}

	fmt.Println("Token inválido: no se pudieron extraer los claims")
	return false, nil
}

// func DecodeTokenRaw

func DecodeToken(tokenString string) (jwt.MapClaims, error) {
	jwtSecret := utils.GetEnv("SECRET_KEY")
	if jwtSecret == "" {
		return nil, errors.New("JWT secret empty")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error al decodificar token: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return claims, fmt.Errorf("token inválido")
	}

	return claims, nil
}

// ResetPassword restablece la contraseña de un usuario.
// Busca al usuario por su correo electrónico y actualiza su contraseña en la base de datos.
// Devuelve un error si el usuario no se encuentra o si ocurre algún problema durante la actualización.
// UserExistsByEmail checks if a user with the given email exists in the database
func UserExistsByEmail(email string) (bool, error) {
	systemUser, err := getSystemUser()
	if err != nil {
		return false, fmt.Errorf("error getting system user: %v", err)
	}

	user := &foundation.User{
		Username: email,
	}

	// Create a base request
	baseRequest, err := foundation.NewBaseRequestWithModel(user, *systemUser)
	if err != nil {
		return false, fmt.Errorf("error creating base request: %v", err)
	}

	// Set up the find options to search by email
	findOptions := user.GetFindOptions(baseRequest)
	findOptions.AddEquals("username", email)
	baseRequest.SetFindOptions(findOptions)

	// Execute the find query
	response := user.Find(baseRequest)
	if response.Error != nil {
		return false, response.Error
	}

	// If we found at least one user with this email, return true
	return response.TotalRows > 0, nil
}

// GetClientIP extracts the client IP from the request headers
func GetClientIP(ipHeader, realIPHeader, remoteAddr string) string {
	// First try X-Forwarded-For (common in many proxies)
	if ipHeader != "" {
		ips := utils.StringToArrayString(ipHeader)
		// Return the first IP that is not a trusted proxy
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" && ip != "unknown" {
				return ip
			}
		}
	}

	// Then try X-Real-IP (used by Nginx)
	if realIPHeader != "" {
		return realIPHeader
	}

	// Finally, use the remote address
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return ip
}

// ProcessPasswordRecovery handles the password recovery process
func ProcessPasswordRecovery(email, xForwardedFor, xRealIP, remoteAddr, userAgent, referer string) error {

	// Verify if the email exists in the database
	exists, err := UserExistsByEmail(email)
	if err != nil {
		return fmt.Errorf("error verifying email: %v", err)
	}

	if !exists {
		ip := GetClientIP(xForwardedFor, xRealIP, remoteAddr)

		// Format the log message
		message := fmt.Sprintf("🚨 **Intento de recuperación de contraseña**\n"+
			"📧 **Email:** %s\n"+
			"🌐 **IP:** %s\n"+
			"🕒 **Hora:** %s\n"+
			"🔗 **Origen:** %s\n"+
			"🖥️ **Navegador/App:** %s",
			email,
			ip,
			time.Now().Format("2006-01-02 15:04:05"),
			referer,
			userAgent,
		)

		log.ToDiscord(log.HookChannelLog, message)
		return fmt.Errorf("user does not exist")
	}

	env := utils.GetEnv("ENVIRONMENT")
	if env == "test" || env == "local" {
		fmt.Println("Skipping email sending in test or local environment")
		return nil
	}

	recoveryToken, err := GenerateRecoveryToken(email)
	if err != nil {
		return fmt.Errorf("error generating recovery token: %v", err)
	}

	if err := SendEmailRecovery(email, recoveryToken); err != nil {
		return fmt.Errorf("error sending recovery email: %v", err)
	}

	return nil
}

func ResetPassword(email string, newPassword string) error {
	model := cloneUserModel()
	model.Username = email

	user, err := getSystemUser()
	if err != nil {
		return err
	}

	request, err := newBaseRequestWithModel(model, *user)
	if err != nil {
		return err
	}

	model, err = model.GetOne(request)
	if err != nil {
		return err
	}

	model.Password = newPassword
	model.ChangePassword = true

	response := model.Update(request)
	if response.Error != nil {
		return response.Error
	}

	return nil

}

func getSystemUser() (*foundation.User, error) {
	adminUser := &foundation.User{}
	return adminUser.GetSystemUser()
}
