package services

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup global
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("SYSTEM_USER", "user_test@weitec.es")
	os.Setenv("SYSTEM_TOKEN", "my_test_system_token")
	os.Setenv("DEFAULT_DATABASE", "weitec_test_db")

	// Ejecutar los tests
	code := m.Run()

	// Teardown global (opcional)
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("SYSTEM_USER")
	os.Unsetenv("SYSTEM_TOKEN")
	os.Unsetenv("DEFAULT_DATABASE")

	// Salir con el código de estado de los tests

	os.Exit(code)
}
