package config

import (
	"crypto/rsa"
	"os"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
)

type Config struct {
	JWTPublicKey       *rsa.PublicKey
	MongoURI           string
	MongoDb            string
	Port               string
	RedisAddress       string
	RedisPassword      string
	CORSAllowedOrigins []string
}

func Load() *Config {

	publicKeyPath := os.Getenv("PUBLIC_KEY_PATH")
	if publicKeyPath == "" {
		publicKeyPath = "/etc/certs/public.pem"
	}
	publicKey, err := loadPublicKey(publicKeyPath)
	if err != nil {
		panic("Failed to load public key: " + err.Error())
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "media"
	}

	redisAddress := os.Getenv("REDIS_ADDRESS")
	if redisAddress == "" {
		redisAddress = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisPassword == "" {
		redisPassword = ""
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if corsOrigins == "" {
		allowedOrigins = []string{"*"} // Default to allow all for development
	} else {
		allowedOrigins = strings.Split(corsOrigins, ",")
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	}

	return &Config{
		JWTPublicKey:       publicKey,
		MongoURI:           mongoURI,
		MongoDb:            dbName,
		Port:               port,
		RedisAddress:       redisAddress,
		RedisPassword:      redisPassword,
		CORSAllowedOrigins: allowedOrigins,
	}
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, err
	}
	return publicKey, nil
}
