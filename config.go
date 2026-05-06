package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// Server
	ServerPort string
	Debug      bool

	// Security
	SessionSecret string
	CSRFSecret    string
}

// AppConfig is the global configuration
var AppConfig Config

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	// Load .env file if exists
	loadEnvFile()

	return Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 3306),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "blog"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
		Debug:      getEnvBool("DEBUG", false),
		SessionSecret: getEnv("SESSION_SECRET", "change-this-secret-in-production"),
		CSRFSecret:    getEnv("CSRF_SECRET", "change-this-csrf-secret-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// InitConfig initializes configuration and logs warnings
func InitConfig() {
	AppConfig = LoadConfig()

	if AppConfig.DBPassword == "" {
		log.Println("鈿狅笍  WARNING: DB_PASSWORD is not set. Database connection may fail.")
	}

	if AppConfig.SessionSecret == "change-this-secret-in-production" {
		log.Println("鈿狅笍  WARNING: Using default SESSION_SECRET. Please set a secure secret in production!")
	}

	if AppConfig.CSRFSecret == "change-this-csrf-secret-in-production" {
		log.Println("鈿狅笍  WARNING: Using default CSRF_SECRET. Please set a secure secret in production!")
	}
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse KEY=VALUE
		equalsIndex := strings.Index(line, "=")
		if equalsIndex > 0 {
			key := strings.TrimSpace(line[:equalsIndex])
			value := strings.TrimSpace(line[equalsIndex+1:])
			os.Setenv(key, value)
		}
	}
}
