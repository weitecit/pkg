package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/weitecit/pkg/foundation"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// MockHttpClient permite simular respuestas de http.Client en los tests
type MockHttpClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// Helper para crear un http.Response simulado
func NewMockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// RoundTripFunc permite usar una función como http.RoundTripper (para http.Client.Transport)
type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func generateTestJWT(claims jwt.MapClaims, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

func TestFillRequestFromToken(t *testing.T) {
	secret := "testsecretkey123"
	os.Setenv("SECRET_KEY", secret)
	defer os.Unsetenv("SECRET_KEY")

	type want struct {
		errContains string
		repoID      string
		spaceID     string
		labels      []string
		language    foundation.Language
		username    string
		userID      interface{}
		contactID   string
		nick        string
	}

	claims := jwt.MapClaims{
		"DomainID":     "domain-xyz",
		"Language":     "es-ES",
		"UserLanguage": "es-ES",
		"SpaceID":      "space-abc",
		"Labels":       []string{"a", "b"},
		"UserID":       "user-1",
		"Username":     "testuser",
		"ContactID":    "contact-1",
		"Nick":         "nicktest",
		"exp":          time.Now().Add(time.Hour).Unix(),
	}
	validToken := generateTestJWT(claims, secret)

	tests := []struct {
		name  string
		token string
		want  want
	}{
		{
			name:  "empty token",
			token: "",
			want:  want{errContains: "token is empty"},
		},
		{
			name:  "invalid token",
			token: "invalid.token.value",
			want:  want{errContains: "SystemService.GetServiceRequestFromToken"},
		},
		{
			name:  "valid token",
			token: validToken,
			want: want{
				errContains: "",
				repoID:      "domain-xyz",
				spaceID:     "space-abc",
				labels:      []string{"a", "b"},
				language:    foundation.Language("es-ES"),
				username:    "testuser",
				//userID:      utils.NewID(),
				contactID: "contact-1",
				nick:      "nicktest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ServiceRequest{Token: tt.token}
			out, err := FillRequestFromToken(req)
			if tt.want.errContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.want.errContains)
				require.Equal(t, req, out)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want.repoID, out.RepoID)
				require.Equal(t, tt.want.spaceID, out.SpaceID)
				require.Equal(t, tt.want.labels, out.Labels)
				require.Equal(t, tt.want.language, out.Language)
				require.Equal(t, tt.want.username, out.User.Username)
				//require.Equal(t, tt.want.userID, out.User.ID)
				require.Equal(t, tt.want.contactID, out.User.ContactID)
				require.Equal(t, tt.want.nick, out.User.Nick)
			}
		})
	}
}

func TestGetExpirationHours(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		unset    bool
		want     int
	}{
		{
			name:  "default (unset)",
			unset: true,
			want:  24,
		},
		{
			name:     "valid env value",
			envValue: "12",
			want:     12,
		},
		{
			name:     "invalid env value",
			envValue: "notanumber",
			want:     0,
		},
		{
			name:     "empty env value",
			envValue: "",
			want:     24,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				os.Unsetenv("TOKEN_EXPIRATION_HOURS")
			} else {
				os.Setenv("TOKEN_EXPIRATION_HOURS", tt.envValue)
				defer os.Unsetenv("TOKEN_EXPIRATION_HOURS")
			}
			require.Equal(t, tt.want, getExpirationHours())
		})
	}
}

func TestCreateWebToken(t *testing.T) {
	secret := "testsecretkey123"
	os.Setenv("SECRET_KEY", secret)
	defer os.Unsetenv("SECRET_KEY")

	t.Run("invalid user", func(t *testing.T) {
		user := foundation.User{} // Usuario sin username ni ID válido
		token, err := CreateWebToken(user)
		require.Error(t, err)
		require.Contains(t, err.Error(), "user is not valid")
		require.Empty(t, token)
	})

	t.Run("valid user", func(t *testing.T) {
		user := foundation.User{
			Username:   "testuser",
			ContactID:  "contact-1",
			Licenses:   []string{"lic1"},
			BaseModel:  foundation.BaseModel{RepoID: "domain-xyz", Language: "es-ES", Labels: &foundation.Labels{"a", "b"}},
			Connection: "conn-1",
			SpaceID:    "space-abc",
			Nick:       "nicktest",
			Roles: foundation.RolePermissions{
				{
					PermissionID:   "perm-1",
					PermissionType: foundation.PermissionTypeEdit,
					Role:           foundation.SpaceRoleAdmin,
				},
			},
		}
		token, err := CreateWebToken(user)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		// Decodificar el token y verificar claims
		parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)
		claims, ok := parsed.Claims.(jwt.MapClaims)
		require.True(t, ok)
		require.Equal(t, "testuser", claims["Username"])
		require.Equal(t, "domain-xyz", claims["DomainID"])
		require.Equal(t, "contact-1", claims["ContactID"])
		require.Equal(t, "es-ES", claims["Language"])
		require.Equal(t, "es-ES", claims["UserLanguage"])
		require.Equal(t, "space-abc", claims["SpaceID"])
		require.Equal(t, "nicktest", claims["Nick"])
		require.ElementsMatch(t, []interface{}{"a", "b"}, claims["UserLabels"])
		require.ElementsMatch(t, []interface{}{"lic1"}, claims["Products"])
		require.Equal(t, "conn-1", claims["Connection"])
		// Verificar roles
		roles, ok := claims["Roles"].([]interface{})
		require.True(t, ok)
		require.Len(t, roles, 1)
		role := roles[0].(map[string]interface{})
		require.Equal(t, "perm-1", role["PermissionID"])
		require.Equal(t, float64(foundation.PermissionTypeEdit), role["PermissionType"])
		require.Equal(t, string(foundation.SpaceRoleAdmin), role["Role"])
	})
}

func TestCreateRefreshToken(t *testing.T) {
	secret := "testsecretkey123"
	os.Setenv("SECRET_KEY", secret)
	defer os.Unsetenv("SECRET_KEY")

	t.Run("userID vacío", func(t *testing.T) {
		token, err := CreateRefreshToken(" ")
		require.Error(t, err)
		require.Contains(t, err.Error(), "userID not provided")
		require.Empty(t, token)
	})

	t.Run("userID válido", func(t *testing.T) {
		userID := "user-123"
		token, err := CreateRefreshToken(userID)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)
		claims, ok := parsed.Claims.(jwt.MapClaims)
		require.True(t, ok)
		require.Equal(t, userID, claims["UserID"])
		require.Contains(t, claims, "exp")
		// exp debe ser un número futuro
		exp, ok := claims["exp"].(float64)
		require.True(t, ok)
		require.Greater(t, int64(exp), time.Now().Unix())
	})
}

func TestUpdateToken(t *testing.T) {
	secret := "testsecretkey123"
	os.Setenv("SECRET_KEY", secret)
	defer os.Unsetenv("SECRET_KEY")

	t.Run("token inválido", func(t *testing.T) {
		token, err := UpdateToken("invalid.token.value")
		require.Error(t, err)
		require.Empty(t, token)
	})

	t.Run("token válido actualiza exp", func(t *testing.T) {
		// Crear un token válido con exp a 1 hora en el futuro
		oldExp := time.Now().Add(1 * time.Hour).Unix()
		claims := jwt.MapClaims{
			"UserID": "user-1",
			"exp":    oldExp,
		}
		token := generateTestJWT(claims, secret)
		updated, err := UpdateToken(token)
		require.NoError(t, err)
		require.NotEmpty(t, updated)

		// Decodificar el token actualizado
		parsed, err := jwt.Parse(updated, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)
		updatedClaims, ok := parsed.Claims.(jwt.MapClaims)
		require.True(t, ok)
		require.Equal(t, "user-1", updatedClaims["UserID"])
		require.Contains(t, updatedClaims, "exp")
		newExp, ok := updatedClaims["exp"].(float64)
		require.True(t, ok)
		require.Greater(t, int64(newExp), oldExp)
		require.Greater(t, int64(newExp), time.Now().Unix())
	})

	t.Run("token expirado da error", func(t *testing.T) {
		// Crear un token expirado
		expiredExp := time.Now().Add(-1 * time.Hour).Unix()
		claims := jwt.MapClaims{
			"UserID": "user-1",
			"exp":    expiredExp,
		}
		token := generateTestJWT(claims, secret)
		updated, err := UpdateToken(token)
		require.Error(t, err)
		require.Empty(t, updated)
	})
}

func TestSendEmail_EnvMissing(t *testing.T) {
	// Limpiar todas las variables de entorno requeridas
	os.Unsetenv("OAUTH_CLIENT_ID")
	os.Unsetenv("OAUTH_CLIENT_SECRET")
	os.Unsetenv("OAUTH_TENANT_ID")
	os.Unsetenv("MICROSOFT_CLIENT")
	os.Unsetenv("LANDING_URI")

	// --- Ejemplo de uso del mock ---
	originalClone := cloneHttpClient
	defer func() { cloneHttpClient = originalClone }()
	cloneHttpClient = func() *http.Client {
		return &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				// Simula cualquier respuesta que quieras aquí
				return NewMockResponse(200, `{"access_token":"mocktoken","expires_in":3600,"token_type":"Bearer"}`), nil
			}),
		}
	}

	err := SendEmailRecovery("test@dominio.com", "token123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error de configuración del servidor")
}

func TestSendEmail_Success(t *testing.T) {
	// Setear variables de entorno requeridas
	os.Setenv("OAUTH_CLIENT_ID", "clientid")
	os.Setenv("OAUTH_CLIENT_SECRET", "secret")
	os.Setenv("OAUTH_TENANT_ID", "tenantid")
	os.Setenv("MICROSOFT_CLIENT", "https://login.microsoftonline.com/%s/oauth2/v2.0/token")
	os.Setenv("LANDING_URI", "https://app.weitec.es/reset/")
	defer func() {
		os.Unsetenv("OAUTH_CLIENT_ID")
		os.Unsetenv("OAUTH_CLIENT_SECRET")
		os.Unsetenv("OAUTH_TENANT_ID")
		os.Unsetenv("MICROSOFT_CLIENT")
		os.Unsetenv("LANDING_URI")
	}()

	// Mock para simular dos respuestas: token y envío de email
	call := 0
	originalClone := cloneHttpClient
	defer func() { cloneHttpClient = originalClone }()
	cloneHttpClient = func() *http.Client {
		return &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				call++
				if call == 1 {
					// Respuesta de autenticación
					return NewMockResponse(200, `{"access_token":"mocktoken","expires_in":3600,"token_type":"Bearer"}`), nil
				}
				// Respuesta de envío de email
				return NewMockResponse(202, ""), nil
			}),
		}
	}

	err := SendEmailRecovery("test@dominio.com", "token123")
	require.NoError(t, err)
}

func TestSendEmail(t *testing.T) {
	// Helper para setear entorno mínimo válido
	setEnv := func() {
		os.Setenv("OAUTH_CLIENT_ID", "clientid")
		os.Setenv("OAUTH_CLIENT_SECRET", "secret")
		os.Setenv("OAUTH_TENANT_ID", "tenantid")
		os.Setenv("MICROSOFT_CLIENT", "https://login.microsoftonline.com/%s/oauth2/v2.0/token")
		os.Setenv("LANDING_URI", "https://app.weitec.es/reset/")
	}
	unsetEnv := func() {
		os.Unsetenv("OAUTH_CLIENT_ID")
		os.Unsetenv("OAUTH_CLIENT_SECRET")
		os.Unsetenv("OAUTH_TENANT_ID")
		os.Unsetenv("MICROSOFT_CLIENT")
		os.Unsetenv("LANDING_URI")
	}

	t.Run("flujo exitoso", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		call := 0
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					call++
					if call == 1 {
						return NewMockResponse(200, `{"access_token":"mocktoken","expires_in":3600,"token_type":"Bearer"}`), nil
					}
					return NewMockResponse(202, ""), nil
				}),
			}
		}
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.NoError(t, err)
	})

	t.Run("error creando request de token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		// Forzar error en NewRequest usando un clientID inválido (URL mal formada)
		os.Setenv("MICROSOFT_CLIENT", "://bad-url")
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error de configuración del servidor")
	})

	t.Run("error haciendo request de token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("fallo http")
				}),
			}
		}
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error de configuración del servidor")
	})

	t.Run("error decodificando token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return NewMockResponse(200, `no es json`), nil
				}),
			}
		}
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error de configuración del servidor")
	})

	t.Run("error en respuesta de autenticación (campo error)", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return NewMockResponse(200, `{"error":"invalid_grant","error_description":"expiró"}`), nil
				}),
			}
		}
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "el token de autenticación ha expirado")
	})

	t.Run("no se recibe access_token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return NewMockResponse(200, `{"access_token":""}`), nil
				}),
			}
		}
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error al obtener token de acceso")
	})

	t.Run("error enviando email", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		call := 0
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					call++
					if call == 1 {
						return NewMockResponse(200, `{"access_token":"mocktoken"}`), nil
					}
					return nil, fmt.Errorf("fallo http email")
				}),
			}
		}
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error al enviar el correo electrónico")
	})

	t.Run("respuesta de email con error (status != 200/202)", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		call := 0
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					call++
					if call == 1 {
						return NewMockResponse(200, `{"access_token":"mocktoken"}`), nil
					}
					return NewMockResponse(400, "bad request"), nil
				}),
			}
		}
		err := SendEmailRecovery("test@dominio.com", "token123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error al enviar el correo electrónico")
	})
}

func TestSendEmailMobile_EnvMissing(t *testing.T) {
	os.Unsetenv("OAUTH_CLIENT_ID")
	os.Unsetenv("OAUTH_CLIENT_SECRET")
	os.Unsetenv("OAUTH_TENANT_ID")
	os.Unsetenv("MICROSOFT_CLIENT")

	err := SendEmailMobile("test@dominio.com", "123456")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error de configuración del servidor")
}

func TestSendEmailMobileResto(t *testing.T) {
	setEnv := func() {
		os.Setenv("OAUTH_CLIENT_ID", "clientid")
		os.Setenv("OAUTH_CLIENT_SECRET", "secret")
		os.Setenv("OAUTH_TENANT_ID", "tenantid")
		os.Setenv("MICROSOFT_CLIENT", "https://login.microsoftonline.com/%s/oauth2/v2.0/token")
	}
	unsetEnv := func() {
		os.Unsetenv("OAUTH_CLIENT_ID")
		os.Unsetenv("OAUTH_CLIENT_SECRET")
		os.Unsetenv("OAUTH_TENANT_ID")
		os.Unsetenv("MICROSOFT_CLIENT")
	}

	t.Run("flujo exitoso", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		call := 0
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					call++
					if call == 1 {
						return NewMockResponse(200, `{"access_token":"mocktoken","expires_in":3600,"token_type":"Bearer"}`), nil
					}
					return NewMockResponse(202, ""), nil
				}),
			}
		}
		err := SendEmailMobile("test@dominio.com", "123456")
		require.NoError(t, err)
	})

	t.Run("error creando request de token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		os.Setenv("MICROSOFT_CLIENT", "://bad-url")
		err := SendEmailMobile("test@dominio.com", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error creando request de token")
	})

	t.Run("error haciendo request de token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("fallo http")
				}),
			}
		}
		err := SendEmailMobile("test@dominio.com", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error obteniendo token")
	})

	t.Run("error decodificando token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return NewMockResponse(200, `no es json`), nil
				}),
			}
		}
		err := SendEmailMobile("test@dominio.com", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error decodificando respuesta de token")
	})

	t.Run("no se recibe access_token", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return NewMockResponse(200, `{"access_token":""}`), nil
				}),
			}
		}
		err := SendEmailMobile("test@dominio.com", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error al obtener token de acceso")
	})

	t.Run("error enviando email", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		call := 0
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					call++
					if call == 1 {
						return NewMockResponse(200, `{"access_token":"mocktoken"}`), nil
					}
					return nil, fmt.Errorf("fallo http email")
				}),
			}
		}
		err := SendEmailMobile("test@dominio.com", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error enviando email")
	})

	t.Run("respuesta de email con error (status != 200/202)", func(t *testing.T) {
		setEnv()
		defer unsetEnv()
		originalClone := cloneHttpClient
		defer func() { cloneHttpClient = originalClone }()
		call := 0
		cloneHttpClient = func() *http.Client {
			return &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
					call++
					if call == 1 {
						return NewMockResponse(200, `{"access_token":"mocktoken"}`), nil
					}
					return NewMockResponse(400, "bad request"), nil
				}),
			}
		}
		err := SendEmailMobile("test@dominio.com", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error del servidor Graph API")
	})
}

func TestGenerateShortCode(t *testing.T) {
	charset := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	cases := []struct {
		name   string
		length int
	}{
		{"zero length", 0},
		{"one char", 1},
		{"four chars", 4},
		{"eight chars", 8},
		{"sixteen chars", 16},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := GenerateShortCode(tc.length)
			require.Equal(t, tc.length, utf8.RuneCountInString(code), "length mismatch")
			for _, c := range code {
				require.Truef(t, strings.ContainsRune(charset, c), "invalid char: %c", c)
			}
		})
	}
}

func TestGenerateRecoveryToken(t *testing.T) {
	secret := "testsecretkey123"
	os.Setenv("SECRET_KEY", secret)
	defer os.Unsetenv("SECRET_KEY")

	email := "test@dominio.com"
	token, err := GenerateRecoveryToken(email)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// El token debe ser válido según ValidateRecoveryToken
	valid, err := ValidateRecoveryToken(token)
	require.NoError(t, err)
	require.True(t, valid, "El token generado debe ser válido")

	// Decodificar y verificar claims
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.Equal(t, email, claims["email"])
	require.Equal(t, "password_recovery", claims["type"])
	exp, ok := claims["exp"].(float64)
	require.True(t, ok)
	require.Greater(t, int64(exp), time.Now().Unix())
}

func TestValidateRecoveryToken(t *testing.T) {
	secret := "testsecretkey123"
	os.Setenv("SECRET_KEY", secret)
	defer os.Unsetenv("SECRET_KEY")

	email := "test@dominio.com"
	// Token válido
	validToken, err := GenerateRecoveryToken(email)
	require.NoError(t, err)
	valid, err := ValidateRecoveryToken(validToken)
	require.NoError(t, err)
	require.True(t, valid, "El token generado debe ser válido")

	// Token con tipo incorrecto
	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"type":  "otro_tipo",
	}
	tokenTipoIncorrecto := generateTestJWT(claims, secret)
	valid, err = ValidateRecoveryToken(tokenTipoIncorrecto)
	require.NoError(t, err)
	require.False(t, valid, "Token con tipo incorrecto no debe ser válido")

	// Token expirado
	claims = jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(-time.Hour).Unix(),
		"type":  "password_recovery",
	}
	tokenExpirado := generateTestJWT(claims, secret)
	valid, err = ValidateRecoveryToken(tokenExpirado)
	require.NoError(t, err)
	require.False(t, valid, "Token expirado no debe ser válido")

	// Token con firma incorrecta
	otroSecret := "otrosecret"
	tokenFirmaIncorrecta := generateTestJWT(jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"type":  "password_recovery",
	}, otroSecret)
	valid, err = ValidateRecoveryToken(tokenFirmaIncorrecta)
	require.NoError(t, err)
	require.False(t, valid, "Token con firma incorrecta no debe ser válido")

	// Token malformado
	valid, err = ValidateRecoveryToken("noesuntoken")
	require.NoError(t, err)
	require.False(t, valid, "Token malformado no debe ser válido")

	// Sin SECRET_KEY en entorno
	os.Unsetenv("SECRET_KEY")
	valid, err = ValidateRecoveryToken(validToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SECRET_KEY no configurado")
	require.False(t, valid)
}

func TestDecodeToken(t *testing.T) {
	secret := "testsecretkey123"
	os.Setenv("SECRET_KEY", secret)
	defer os.Unsetenv("SECRET_KEY")

	email := "test@dominio.com"
	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"type":  "password_recovery",
	}
	// Token válido
	token := generateTestJWT(claims, secret)
	outClaims, err := DecodeToken(token)
	require.NoError(t, err)
	if outClaims != nil {
		require.Equal(t, email, outClaims["email"])
		require.Equal(t, "password_recovery", outClaims["type"])
	}

	// Token expirado
	expiredClaims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(-time.Hour).Unix(),
		"type":  "password_recovery",
	}

	expiredToken := generateTestJWT(expiredClaims, secret)
	claims, err = DecodeToken(expiredToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error al decodificar token")

	// Token con firma incorrecta
	otroSecret := "otrosecret"
	badClaims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"type":  "password_recovery",
	}
	badToken := generateTestJWT(badClaims, otroSecret)
	claims, err = DecodeToken(badToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error al decodificar token")

	// Token malformado
	claims, err = DecodeToken("noesuntoken")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error al decodificar token")

	// Sin SECRET_KEY
	os.Unsetenv("SECRET_KEY")
	claims, err = DecodeToken(token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "JWT secret empty")
}
