package log

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBLogger struct {
	client     *mongo.Client
	database   string
	collection string
}

// LogType represents the type of log entry
type LogType string

const (
	// LogTypeSystem represents system logs
	LogTypeSystem LogType = "system"
	// LogTypeAuth represents authentication logs
	LogTypeAuth LogType = "auth"
	// LogTypeAPI represents API access logs
	LogTypeAPI LogType = "api"
)

// LogEntry represents a log entry in MongoDB
type LogEntry struct {
	Level     string    `bson:"level"`
	Type      LogType   `bson:"type"`
	Message   string    `bson:"message"`
	Error     string    `bson:"error,omitempty"`
	Timestamp time.Time `bson:"timestamp"`
	Fields    bson.M    `bson:"fields,omitempty"`
}

// NewMongoDBLogger creates a new MongoDB logger using an existing MongoDB client
func NewMongoDBLogger(client *mongo.Client, database, collection string) *MongoDBLogger {
	return &MongoDBLogger{
		client:     client,
		database:   database,
		collection: collection,
	}
}

// Write implements io.Writer interface
func (m *MongoDBLogger) Write(p []byte) (n int, err error) {
	logMessage := string(p)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar la conexión
	err = m.client.Ping(ctx, nil)
	if err != nil {
		return len(p), nil // No retornar error para no detener el logging
	}

	// Verificar que la base de datos existe
	dbNames, err := m.client.ListDatabaseNames(ctx, bson.M{"name": m.database})
	if err != nil {
		return len(p), nil
	}

	dbExists := false
	for _, name := range dbNames {
		if name == m.database {
			dbExists = true
			break
		}
	}

	if !dbExists {
		return len(p), nil
	}

	// Crear un mapa para almacenar los campos del log
	logEntry := bson.M{
		"message":   logMessage,
		"level":     "info", // Valor por defecto
		"type":      string(LogTypeSystem),
		"timestamp": time.Now(),
	}

	// Intentar analizar el mensaje como JSON
	var jsonData map[string]interface{}
	if err := bson.UnmarshalExtJSON(p, false, &jsonData); err == nil {
		// Si es un JSON válido, extraer campos relevantes
		if level, ok := jsonData["level"].(string); ok {
			logEntry["level"] = level
		}
		if msg, ok := jsonData["message"].(string); ok {
			logEntry["message"] = msg
		}
		if logType, ok := jsonData["type"].(string); ok {
			logEntry["type"] = logType
		}
		if errMsg, ok := jsonData["error"].(string); ok {
			logEntry["error"] = errMsg
		}

		// Agregar campos adicionales
		for k, v := range jsonData {
			if k != "level" && k != "message" && k != "type" && k != "error" {
				logEntry[k] = v
			}
		}
	}

	// Insertar en MongoDB
	db := m.client.Database(m.database)
	collection := db.Collection(m.collection)

	// Verificar si la colección existe
	names, err := db.ListCollectionNames(ctx, bson.M{"name": m.collection})
	if err != nil {
		return len(p), nil
	}

	if len(names) == 0 {
		// La colección no existe, crearla
		db.CreateCollection(ctx, m.collection)
	}

	// Insertar el log
	_, err = collection.InsertOne(ctx, logEntry)
	if err != nil {
		return len(p), nil
	}

	return len(p), nil
}

// Close closes the MongoDB connection
func (m *MongoDBLogger) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

// getCurrentDayRange returns the start and end of the current day in UTC
func (m *MongoDBLogger) getCurrentDayRange() (time.Time, time.Time) {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)
	return startOfDay, endOfDay
}

// GetLogs retrieves log entries from MongoDB with optional filtering
func (m *MongoDBLogger) GetLogs(filter bson.M, limit int, skip int) ([]LogEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set default values if not provided
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Add current day filter
	if filter == nil {
		filter = bson.M{}
	}

	// Solo añadir rango del día actual si no hay timestamp en el filtro
	if _, ok := filter["timestamp"]; !ok {
		startOfDay, endOfDay := m.getCurrentDayRange()
		filter["timestamp"] = bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		}
	}

	// Create options for sorting by timestamp (newest first) and pagination
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(skip))

	// Get collection
	collection := m.client.Database(m.database).Collection(m.collection)

	// Execute query
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode results
	var logs []LogEntry
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

// GetLogsByLevel retrieves logs filtered by level (case-insensitive)
func (m *MongoDBLogger) GetLogsByLevel(level string, limit int, skip int) ([]LogEntry, error) {
	filter := bson.M{"level": bson.M{"$regex": "^" + level + "$", "$options": "i"}}
	return m.GetLogs(filter, limit, skip)
}

// CountLogsByLevel counts logs filtered by level (case-insensitive)
func (m *MongoDBLogger) CountLogsByLevel(level string, dateStr string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := m.client.Database(m.database).Collection(m.collection)

	// Parseamos la fecha recibida, si no se recibe usamos el día actual
	var startOfDay, endOfDay time.Time
	if dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr) // Formato YYYY-MM-DD
		if err != nil {
			return 0, fmt.Errorf("fecha inválida: %w", err)
		}
		startOfDay = parsedDate
		endOfDay = parsedDate.Add(24 * time.Hour)
	} else {
		startOfDay, endOfDay = m.getCurrentDayRange()
	}

	// Filtro
	filter := bson.M{
		"timestamp": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}

	if level != "all" {
		filter["level"] = bson.M{"$regex": "^" + level + "$", "$options": "i"}
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error counting logs: %w", err)
	}

	return count, nil
}

// GetLogsByType retrieves logs filtered by type
func (m *MongoDBLogger) GetLogsByType(logType LogType, limit int, skip int) ([]LogEntry, error) {
	filter := bson.M{"type": logType}
	return m.GetLogs(filter, limit, skip)
}

// GetLogsByTimeRange retrieves logs within a specific time range
func (m *MongoDBLogger) GetLogsByTimeRange(start, end time.Time, limit int, skip int) ([]LogEntry, error) {
	filter := bson.M{
		"timestamp": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}
	return m.GetLogs(filter, limit, skip)
}
