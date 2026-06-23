package main

import (
	"fmt"
	"log"

	"bookmyvenue/config"
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/handler"
	"bookmyvenue/internal/repository"
	"bookmyvenue/internal/router"
	"bookmyvenue/internal/service"
)

func main() {
	cfg := config.LoadConfig()

	db := config.ConnectDB(cfg)

	err := db.AutoMigrate(&domain.User{}, &domain.Admin{})
	if err != nil {
		log.Fatalf("Auto-migration failed: %v", err)
	}

	// User & Owner
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg)
	authHandler := handler.NewAuthHandler(authService)

	// Admin
	adminRepo := repository.NewAdminRepository(db)
	adminAuthService := service.NewAdminAuthService(adminRepo, cfg)
	adminAuthHandler := handler.NewAdminAuthHandler(adminAuthService)

	r := router.SetupRouter(cfg, authHandler,adminAuthHandler)

	port := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("🚀 BookMyVenue server starting on port %s", cfg.ServerPort)
	if err := r.Run(port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}