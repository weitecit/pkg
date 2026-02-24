package foundation

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestRolePermissionMapping verifica cómo se mapea rolepermission desde MongoDB
func TestRolePermissionMapping() {
	// Conectar a MongoDB
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// Obtener la colección de usuarios
	collection := client.Database("your_database").Collection("users")

	// Buscar un usuario específico
	var result bson.M
	err = collection.FindOne(context.Background(), bson.M{"username": "pruebaweitec@gmail.com"}).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Documento RAW de MongoDB ===")
	fmt.Printf("%+v\n\n", result)

	// Verificar si rolepermission existe
	if rolePermData, ok := result["rolepermission"]; ok {
		fmt.Println("=== Campo rolepermission encontrado ===")
		fmt.Printf("Tipo: %T\n", rolePermData)
		fmt.Printf("Valor: %+v\n\n", rolePermData)

		// Intentar convertir a map
		if rolePermMap, ok := rolePermData.(bson.M); ok {
			fmt.Println("=== Campos dentro de rolepermission ===")
			for key, value := range rolePermMap {
				fmt.Printf("%s: %v (tipo: %T)\n", key, value, value)
			}
		}
	} else {
		fmt.Println("⚠️ Campo rolepermission NO encontrado en el documento")
	}

	// Ahora intentar deserializar a struct User
	fmt.Println("\n=== Intentando deserializar a struct User ===")
	var user User
	err = collection.FindOne(context.Background(), bson.M{"username": "pruebaweitec@gmail.com"}).Decode(&user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Username: %s\n", user.Username)
	fmt.Printf("RolePermission.PermissionID: %s\n", user.RolePermission.PermissionID)
	fmt.Printf("RolePermission.PermissionType: %d\n", user.RolePermission.PermissionType)
	fmt.Printf("RolePermission.Role: %s\n", user.RolePermission.Role)
}
