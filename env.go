package main

import (
	"os"

	"github.com/joho/godotenv"
)

func OpenEnv() error {
	return godotenv.Load(".env")
}

func IsDevelopment() bool {
	return os.Getenv("ENV") == "development"
}

func IsProduction() bool {
	return os.Getenv("ENV") == "production"
}
