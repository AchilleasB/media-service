package config

import (
	"crypto/rsa"
	"os"

	"github.com/golang-jwt/jwt/v4"
)

type Config struct {
	JWTPublicKey  *rsa.PublicKey
	MongoURI      string
	Port          string
	RedisAddress  string
	RedisPassword string
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
		port = "8081"
	}

	return &Config{
		JWTPublicKey:  publicKey,
		MongoURI:      mongoURI,
		Port:          port,
		RedisAddress:  redisAddress,
		RedisPassword: redisPassword,
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
