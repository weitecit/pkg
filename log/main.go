package log

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// OutputType defines the type of logging output
type OutputType string

const (
	// ConsoleOutput sends logs to console
	ConsoleOutput OutputType = "console"
	// MongoDBOutput sends logs to MongoDB
	MongoDBOutput OutputType = "mongodb"
)

var (
	// DefaultOutputs specifies the default logging outputs
	DefaultOutputs = []OutputType{ConsoleOutput}
	outputWriters  []io.Writer
	loggerMutex    sync.Mutex
)

// Config holds the logging configuration
type Config struct {
	Level           zerolog.Level
	Outputs         []OutputType
	MongoClient     *mongo.Client // Cliente de MongoDB existente
	MongoDatabase   string
	MongoCollection string
}

// Init initializes the logging system with the specified configuration
func Init(cfg Config) error {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	// Reset output writers
	outputWriters = []io.Writer{}

	// Configure outputs
	for _, output := range cfg.Outputs {
		switch output {
		case ConsoleOutput:
			outputWriters = append(outputWriters, zerolog.ConsoleWriter{Out: os.Stderr})
		case MongoDBOutput:
			if cfg.MongoClient != nil && cfg.MongoDatabase != "" && cfg.MongoCollection != "" {
				mongoLogger := NewMongoDBLogger(cfg.MongoClient, cfg.MongoDatabase, cfg.MongoCollection)
				outputWriters = append(outputWriters, mongoLogger)
			} else {
				fmt.Fprintf(os.Stderr, "[Logger] MongoDB client not configured or missing database/collection\n")
			}
		}
	}

	// If no outputs are specified, use console as default
	if len(outputWriters) == 0 {
		outputWriters = append(outputWriters, zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Create multi-writer for all outputs
	multiWriter := zerolog.MultiLevelWriter(outputWriters...)
	log.Logger = log.Output(multiWriter)

	// Set log level
	zerolog.SetGlobalLevel(cfg.Level)
	fmt.Fprintf(os.Stderr, "[Logger] Logging initialized with level: %s\n", cfg.Level)

	return nil
}

// InitWithDefaults initializes the logger with default console output
func InitWithDefaults(logLevel byte) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevelName := zerolog.Level(logLevel)

	err := Init(Config{
		Level:   logLevelName,
		Outputs: DefaultOutputs,
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
}

// AddOutput adds a new output writer to the logger
func AddOutput(writer io.Writer) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	outputWriters = append(outputWriters, writer)
	multiWriter := zerolog.MultiLevelWriter(outputWriters...)
	log.Logger = log.Output(multiWriter)
}

// RemoveOutput removes an output writer from the logger
func RemoveOutput(writer io.Writer) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	for i, w := range outputWriters {
		if w == writer {
			outputWriters = append(outputWriters[:i], outputWriters[i+1:]...)
			break
		}
	}

	if len(outputWriters) == 0 {
		// Fallback to console if no outputs remain
		outputWriters = append(outputWriters, zerolog.ConsoleWriter{Out: os.Stderr})
	}

	multiWriter := zerolog.MultiLevelWriter(outputWriters...)
	log.Logger = log.Output(multiWriter)
}

func Fatal(err error) {
	log.Fatal().Err(err).Send()
}

func Err(err error) {
	log.Error().Err(err).Send()
}

func Errf(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}
func Warn(err error) {
	log.Warn().Err(err).Send()
}

func Warnf(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

func Debug(err error) {
	log.Debug().Err(err).Send()
}

func Debugf(format string, v ...interface{}) {
	log.Debug().Msgf(format, v...)
}

func Print(msg string) {
	log.Print(msg)
}

func Trace(err error) {
	log.Trace().Err(err).Send()
}

func Tracef(format string, v ...interface{}) {
	log.Trace().Msgf(format, v...)
}

func Info(err error) {
	log.Info().Err(err).Send()
}

func Infof(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

func Panic(err error) {
	log.Panic().Err(err).Send()
}

func Log(text string) {
	log.Trace().Msg(text)
}

func ToDiscord(channel HookChannel, text string) error {

	discordUser := "🤖 Minion"
	content := text

	message := Message{
		Username: &discordUser,
		Content:  &content,
	}

	return sendDiscordMessage(channel, message)
}
