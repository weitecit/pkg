package testutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewTestMongoServer prueba la creación de un servidor de prueba
func TestNewTestMongoServer(t *testing.T) {
	server, err := NewTestMongoServer("testdb")
	require.NoError(t, err, "No se pudo crear el servidor de prueba")
	defer server.Close()

	// Verificar que podemos hacer ping a la base de datos
	err = server.Client.Ping(context.Background(), nil)
	require.NoError(t, err, "No se pudo hacer ping a la base de datos")

	// Verificar que la URI no está vacía
	require.NotEmpty(t, server.URI, "La URI del servidor no debe estar vacía")
}
