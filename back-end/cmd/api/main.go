package main

import (
	"log"

	"bookmyvenue/config"
	"bookmyvenue/internal/domain"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	db := config.ConnectDB(cfg)

	err := db.AutoMigrate(&domain.User{}, &domain.Admin{})
	if err != nil {
		log.Fatalf("Auto-migration failed: %v", err)
	}

	r := gin.Default()

	log.Println("Starting Gin HTTP server on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}