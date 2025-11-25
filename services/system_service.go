package services

import (
	"bytes"
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
)

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
	for _, permission := range user.Roles {
		userRole := UserRole{
			PermissionID:   permission.PermissionID,
			PermissionType: float64(permission.PermissionType),
			Role:           string(permission.Role),
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

// SendEmail envía un correo electrónico de recuperación de contraseña al destinatario especificado.
// Utiliza Microsoft Graph API para enviar el correo electrónico y notifica a Discord sobre el estado de la operación.
// Devuelve un error genérico si ocurre algún problema durante el proceso.
func SendEmail(to string, recoveryToken string) error {
	// Obtener configuración de entorno
	from := "it@weitec.es"
	clientID := utils.GetEnv("OAUTH_CLIENT_ID")
	clientSecret := utils.GetEnv("OAUTH_CLIENT_SECRET")
	tenantID := utils.GetEnv("OAUTH_TENANT_ID")
	microsoftClient := utils.GetEnv("MICROSOFT_CLIENT")
	landingPage := utils.GetEnv("LANDING_URI")

	// Validar variables de entorno requeridas
	if clientID == "" || clientSecret == "" || tenantID == "" || microsoftClient == "" || landingPage == "" {
		errMsg := fmt.Sprintf("Faltan variables de entorno requeridas: OAUTH_CLIENT_ID=%t, OAUTH_TENANT_ID=%t, MICROSOFT_CLIENT=%t, LANDING_URI=%t",
			clientID != "", tenantID != "", microsoftClient != "", landingPage != "")
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error de configuración del servidor") // Mensaje genérico para el frontend
	}

	tokenURL := fmt.Sprintf(microsoftClient, tenantID)

	// Preparar datos para la solicitud de token
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("grant_type", "client_credentials")

	// Crear y enviar solicitud de token
	tokenReq, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		errMsg := fmt.Sprintf("Error creando request de token: %v", err)
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error de configuración del servidor") // Mensaje genérico para el frontend
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Configurar cliente HTTP con timeout
	client := cloneHttpClient()

	// Enviar la solicitud de token
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		errMsg := fmt.Sprintf("Error al realizar la solicitud de token: %v", err)
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		fmt.Printf("[DEBUG] Error en la solicitud de token: %v\n", err)
		return fmt.Errorf("error de configuración del servidor") // Mensaje genérico para el frontend
	}
	defer tokenResp.Body.Close()

	// Leer el cuerpo de la respuesta para tenerlo disponible en caso de error
	bodyBytes, _ := io.ReadAll(tokenResp.Body)
	fmt.Printf("[DEBUG] Respuesta de autenticación (Status: %d): %s\n", tokenResp.StatusCode, string(bodyBytes))

	var tokenData struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}

	// Decodificar la respuesta
	if err := json.Unmarshal(bodyBytes, &tokenData); err != nil {
		errMsg := fmt.Sprintf("Error decodificando respuesta de token: %v. Respuesta: %s", err, string(bodyBytes))
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error de configuración del servidor") // Mensaje genérico para el frontend
	}

	// Verificar si hay un error en la respuesta
	if tokenData.Error != "" {
		errMsg := fmt.Sprintf("Error en la respuesta de autenticación: %s - %s", tokenData.Error, tokenData.Description)

		// Log detallado para depuración
		fmt.Printf("[DEBUG] Error de autenticación: %+v\n", tokenData)

		// Verificar específicamente si el token expiró
		if strings.Contains(strings.ToLower(tokenData.Description), "expir") ||
			tokenData.Error == "invalid_grant" {
			errMsg = "⚠️ ATENCIÓN: El token de autenticación ha expirado. Se requiere renovación de credenciales."
			log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
			return fmt.Errorf("el token de autenticación ha expirado, por favor contacte con soporte")
		}

		// Otros errores de autenticación
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		fmt.Printf("[DEBUG] Error detallado: %s\n", errMsg)
		return fmt.Errorf("error de autenticación con el servicio de correo")
	}

	if tokenData.AccessToken == "" {
		errMsg := fmt.Sprintf("No se recibió token de acceso. Respuesta: %s", string(bodyBytes))
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error al obtener token de acceso")
	}

	resetURL := fmt.Sprintf(landingPage+"%s", recoveryToken)
	println("•••••••••••••••••••••••••••••••••")
	println("resetURL", resetURL)
	println("•••••••••••••••••••••••••••••••••")
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

	graphEndpoint := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", from)
	emailData := map[string]interface{}{
		"message": map[string]interface{}{
			"subject":    "Recuperación de contraseña - Weitec",
			"importance": "high",
			"body": map[string]interface{}{
				"contentType": "HTML",
				"content":     body,
			},
			"toRecipients": []map[string]interface{}{
				{
					"emailAddress": map[string]string{
						"address": to,
					},
				},
			},
		},
		"saveToSentItems": true,
	}

	// Crear el JSON para el correo electrónico
	jsonData, err := json.Marshal(emailData)
	if err != nil {
		errMsg := fmt.Sprintf("Error creando JSON para el correo: %v", err)
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error al enviar el correo electrónico")
	}

	// Crear la solicitud para enviar el correo
	mailReq, err := http.NewRequest("POST", graphEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		errMsg := fmt.Sprintf("Error creando solicitud de envío de correo: %v", err)
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error al enviar el correo electrónico")
	}

	// Configurar encabezados de la solicitud
	mailReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenData.AccessToken))
	mailReq.Header.Set("Content-Type", "application/json")

	// Enviar la solicitud
	log.ToDiscord(log.HookChannelLog, fmt.Sprintf("📤 Enviando correo de recuperación a: %s", to))
	resp, err := client.Do(mailReq)
	if err != nil {
		errMsg := fmt.Sprintf("Error al enviar el correo electrónico: %v", err)
		log.ToDiscord(log.HookChannelLog, "❌ Error en SendEmail: "+errMsg)
		return fmt.Errorf("error al enviar el correo electrónico")
	}
	defer resp.Body.Close()

	// Leer el cuerpo de la respuesta para registrarlo en caso de error
	respBody, _ := io.ReadAll(resp.Body)

	// Verificar el código de estado de la respuesta
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		// Enviar notificación de error a Discord con más detalles
		discordMsg := fmt.Sprintf("❌ Error al enviar correo a %s:\nCódigo: %d\nRespuesta: %s\nHeaders: %v",
			to, resp.StatusCode, string(respBody), resp.Header)
		if len(discordMsg) > 1900 { // Limitar la longitud del mensaje para Discord
			discordMsg = discordMsg[:1900] + "..."
		}
		log.ToDiscord(log.HookChannelLog, discordMsg)

		// Devolver un mensaje genérico al frontend
		return fmt.Errorf("error al enviar el correo electrónico")
	}

	// Éxito: notificar a Discord
	successMsg := fmt.Sprintf("✅ Correo de recuperación enviado exitosamente a: %s", to)
	fmt.Println(successMsg)
	log.ToDiscord(log.HookChannelLog, successMsg)
	return nil
}

func SendEmailMobile(to, recoveryCode string) error {
	from := "it@weitec.es"
	clientID := utils.GetEnv("OAUTH_CLIENT_ID")
	clientSecret := utils.GetEnv("OAUTH_CLIENT_SECRET")
	tenantID := utils.GetEnv("OAUTH_TENANT_ID")
	microsoftClient := utils.GetEnv("MICROSOFT_CLIENT")
	tokenURL := fmt.Sprintf(microsoftClient, tenantID)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("grant_type", "client_credentials")

	tokenReq, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creando request de token: %v", err)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := cloneHttpClient()
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("error obteniendo token: %v", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return fmt.Errorf("error decodificando respuesta de token: %v", err)
	}

	if tokenData.AccessToken == "" {
		bodyBytes, _ := io.ReadAll(tokenResp.Body)
		return fmt.Errorf("no se recibió token de acceso. Respuesta: %s", string(bodyBytes))
	}

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

	graphEndpoint := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", from)
	emailData := map[string]interface{}{
		"message": map[string]interface{}{
			"subject":    "Código de recuperación - Weitec",
			"importance": "high",
			"body": map[string]interface{}{
				"contentType": "HTML",
				"content":     body,
			},
			"toRecipients": []map[string]interface{}{
				{
					"emailAddress": map[string]string{
						"address": to,
					},
				},
			},
		},
		"saveToSentItems": true,
	}

	jsonData, err := json.Marshal(emailData)
	if err != nil {
		return fmt.Errorf("error creando JSON: %v", err)
	}

	mailReq, err := http.NewRequest("POST", graphEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creando request: %v", err)
	}

	mailReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenData.AccessToken))
	mailReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(mailReq)
	if err != nil {
		return fmt.Errorf("error enviando email: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error del servidor Graph API (%d): %s\nHeaders: %v",
			resp.StatusCode, string(bodyBytes), resp.Header)
	}

	fmt.Printf("Email de recuperación enviado exitosamente a: %s\n", to)
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

	if err := SendEmail(email, recoveryToken); err != nil {
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
