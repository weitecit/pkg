# Weitec GO Packages

## 1. Foundation Package
The **`foundation`** package serves as the core framework for the application's domain layer and data access. It provides base structures and interfaces to standardize how data models are defined and how they interact with the database (specifically MongoDB).

**Key Features:**
*   **`BaseModel`**: A struct designed to be embedded in your domain models. It provides common fields like `ID`, `CreatedBy`, `UpdatedBy`, `DeletedBy` (soft delete), `Labels`, and `Tags`.
*   **Repository Pattern**: Defines a generic `Repository` interface and a concrete `MongoRepository` implementation for standard CRUD operations (`Find`, `FindOne`, `Update`, `Delete`).
*   **`FindOptions`**: A powerful struct to build complex database queries with filters, sorting, and pagination without writing raw MongoDB queries.

**Basic Usage Example:**

```go
package main

import (
    "fmt"
    "github.com/weitecit/pkg/foundation"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// 1. Define your model embedding BaseModel
type Product struct {
    foundation.BaseModel `bson:",inline"`
    Name                 string  `json:"name" bson:"name"`
    Price                float64 `json:"price" bson:"price"`
}

// Implement required interface methods (boilerplate usually handled by the package)
func (p *Product) GetCollection() (string, bool) { return "products", false }

func main() {
    // 2. Initialize the repository
    repo, _ := foundation.NewRepository("mongodb://localhost:27017", foundation.RepoTypeMongoDB, "my_db", "products", false)

    // 3. Create a request to find a product
    request := foundation.RepoRequest{
        Model: &Product{},
        FindOptions: *foundation.NewFindOptions(),
    }
    
    // Add a filter: WHERE name = "Laptop"
    request.FindOptions.AddEquals("name", "Laptop")

    // 4. Execute the query
    response := repo.FindOne(request)
    
    if response.Error != nil {
        fmt.Println("Error:", response.Error)
    } else {
        product := request.Model.(*Product)
        fmt.Printf("Found product: %s (ID: %s)\n", product.Name, product.GetIDStr())
    }
}
```

## 2. Log Package
The **`log`** package provides a centralized logging system based on `zerolog`. It supports multiple output destinations simultaneously, such as the console and MongoDB, and includes integration for sending alerts to Discord.

**Key Features:**
*   **Multi-Output**: Can write logs to Console, MongoDB, or both.
*   **Levels**: Supports standard levels (`Info`, `Warn`, `Err`, `Debug`, `Trace`).
*   **Discord Integration**: Helper function `ToDiscord` to send messages to specific channels.

**Basic Usage Example:**

```go
package main

import (
    "github.com/weitecit/pkg/log"
)

func main() {
    // 1. Initialize the logger (usually done at app startup)
    // Simple console init:
    log.InitWithDefaults(0) 

    // Or with specific config:
    // log.Init(log.Config{Level: zerolog.InfoLevel, Outputs: []log.OutputType{log.ConsoleOutput}})

    // 2. Log messages
    log.Info(nil) // Just logs a timestamp/context if err is nil, or use Infof
    log.Infof("Application started on port %d", 8080)

    // 3. Log an error
    err := doSomething()
    if err != nil {
        log.Err(err) // Logs the error with stack trace context
    }
}

func doSomething() error {
    return nil
}
```

## 3. Utils Package
The **`utils`** package is a comprehensive collection of helper functions designed to handle common tasks and avoid code duplication. It covers date manipulation, string processing, type conversion, and encryption.

**Key Features:**
*   **Date/Time**: Helpers like `Now()`, `StringToTime()`, `DateToStr()`, and `ReformatDate()`.
*   **String Manipulation**: `Normalize()` (removes accents/spaces), `ToSnakeCase()`, `RemovePunctuation()`.
*   **Type Conversion**: Safe conversions like `StrToInt()`, `StringToFloat64()`, `StringToBoolean()`.
*   **Security**: Simple AES `Encrypt` and `Decrypt` functions.

**Basic Usage Example:**

```go
package main

import (
    "fmt"
    "github.com/weitecit/pkg/utils"
)

func main() {
    // String Normalization (useful for search or IDs)
    rawName := "Crème Brûlée"
    cleanName := utils.Normalize(rawName) 
    fmt.Println(cleanName) // Output: cremebrulee

    // Date Conversion
    dateStr := "2023-11-19"
    datePtr, err := utils.StringToTime(dateStr, false)
    if err == nil {
        fmt.Println("Date object:", datePtr)
    }

    // Type Conversion
    number := utils.StrToInt("123")
    fmt.Println("Integer:", number + 10) // Output: 133

    // MongoDB ID Generation
    newID := utils.NewIDHex()
    fmt.Println("New Mongo ID:", newID)
}
```