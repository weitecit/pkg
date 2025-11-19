package testutils

import (
	"context"
	"fmt"
	"os"

	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestMongoServer representa un servidor MongoDB en memoria para pruebas
type TestMongoServer struct {
	server   *memongo.Server
	Client   *mongo.Client
	Database *mongo.Database
	URI      string
}

// NewTestMongoServer crea una nueva instancia de MongoDB para pruebas.
// Si MONGO_REPO es "memory", usará MongoDB en memoria.
// De lo contrario, usará la URL especificada en MONGO_REPO (por defecto "mongodb://localhost:27017/").
func NewTestMongoServer(dbName string) (*TestMongoServer, error) {
	var (
		server *memongo.Server
		uri   string
		err   error
	)

	// Verificar si debemos usar MongoDB en memoria o una instancia local
	mongoRepo := os.Getenv("MONGO_REPO")
	if mongoRepo == "memory" {
		// Configurar el servidor de MongoDB en memoria
		mongoVersion := "4.4.24"
		
		// Configurar opciones de memongo
		opts := &memongo.Options{
			MongoVersion: mongoVersion,
			// Usar el directorio temporal del sistema para los binarios
			DownloadURL: "https://fastdl.mongodb.org/osx/mongodb-macos-x86_64-4.4.24.tgz",
		}

		server, err = memongo.StartWithOptions(opts)
		if err != nil {
			return nil, fmt.Errorf("no se pudo iniciar MongoDB en memoria: %w", err)
		}
		uri = server.URI()
	} else {
		// Usar una instancia local de MongoDB
		if mongoRepo == "" {
			uri = "mongodb://localhost:27017/"
		} else {
			uri = mongoRepo
		}
	}

	// Conectar al servidor
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		if server != nil {
			server.Stop()
		}
		return nil, fmt.Errorf("no se pudo conectar a MongoDB: %w", err)
	}

	// Verificar la conexión
	err = client.Ping(context.Background(), nil)
	if err != nil {
		client.Disconnect(context.Background())
		server.Stop()
		return nil, fmt.Errorf("no se pudo hacer ping a MongoDB: %w", err)
	}

	db := client.Database(dbName)

	return &TestMongoServer{
		server:   server,
		Client:   client,
		Database: db,
		URI:      uri,
	}, nil
}

// Close detiene el servidor y cierra la conexión
func (s *TestMongoServer) Close() {
	if s.Client != nil {
		s.Client.Disconnect(context.Background())
	}
	if s.server != nil {
		s.server.Stop()
	}
}

// CleanDB elimina todas las colecciones de la base de datos
func (s *TestMongoServer) CleanDB() error {
	if s.Database == nil {
		return nil
	}

	// Obtener todas las colecciones
	collections, err := s.Database.ListCollectionNames(context.Background(), map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("error al listar colecciones: %w", err)
	}

	// Eliminar cada colección
	for _, name := range collections {
		err = s.Database.Collection(name).Drop(context.Background())
		if err != nil {
			return fmt.Errorf("error al eliminar colección %s: %w", name, err)
		}
	}

	return nil
}
