package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	// Auth
	JWTSecret             string
	AccessTokenExpiryMins int
	RefreshTokenExpiryDays int
	// Server
	ServerPort string
}

func LoadConfig() *Config {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbName == "" {
		dbName = "bookmyvenue"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	if dbPassword == "" {
		log.Fatal("CRITICAL CONFIGURATION ERROR: DB_PASSWORD is not set in the environment variables!")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("CRITICAL: JWT_SECRET is not set!")
	}

	accessExpiry := getEnvAsInt("ACCESS_TOKEN_EXPIRY_MINS", 15)
	refreshExpiry := getEnvAsInt("REFRESH_TOKEN_EXPIRY_DAYS", 30)

	serverPort := getEnvOrDefault("SERVER_PORT", "8080")


	return &Config{
		DBHost:     dbHost,
		DBPort:     dbPort,
		DBUser:     dbUser,
		DBPassword: dbPassword,
		DBName:     dbName,
		DBSSLMode:  dbSSLMode,
		JWTSecret: jwtSecret,
		AccessTokenExpiryMins: accessExpiry,
		RefreshTokenExpiryDays: refreshExpiry,
		ServerPort: serverPort,
	}
}


func getEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		log.Printf("WARNING: %s is not a valid integer, using default %d", key, defaultVal)
		return defaultVal
	}
	return val
}