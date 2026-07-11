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
	"bookmyvenue/pkg/s3"
)

func main() {
	cfg := config.LoadConfig()

	db := config.ConnectDB(cfg)
	rdb := config.ConnectRedis(cfg)

	err := db.AutoMigrate(&domain.User{},
		&domain.Admin{},
		&domain.Venue{},
		&domain.Space{},
		&domain.Slot{},
		&domain.Booking{},
		&domain.Payment{},
		&domain.PaymentAuditLog{},
		&domain.Refund{},
		&domain.CancellationPolicy{},
		&domain.VenueEditDraft{},
		&domain.OutboxEvent{},)

	if err != nil {
		log.Fatalf("Auto-migration failed: %v", err)
	}

	// S3 Client
	s3Client, err := s3.NewS3Client(cfg)
	if err != nil {
		log.Printf("⚠️  S3 Client not initialized: %v (presigned URLs won't work)", err)
	}

	// User & Owner
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg)
	authHandler := handler.NewAuthHandler(authService)

	// Admin
	adminRepo := repository.NewAdminRepository(db)
	adminAuthService := service.NewAdminAuthService(adminRepo, cfg)
	adminAuthHandler := handler.NewAdminAuthHandler(adminAuthService)

	// Venue
	venueRepo := repository.NewVenueRepository(db)
	spaceRepo := repository.NewSpaceRepository(db)
	venueService := service.NewVenueService(venueRepo, spaceRepo, s3Client,rdb)
	venueHandler := handler.NewVenueHandler(venueService)

	adminVenueService := service.NewAdminVenueService(venueRepo)
	adminVenueHandler := handler.NewAdminVenueHandler(adminVenueService)

	r := router.SetupRouter(cfg,rdb, authHandler,adminAuthHandler,venueHandler,adminVenueHandler)

	port := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("🚀 BookMyVenue server starting on port %s", cfg.ServerPort)
	if err := r.Run(port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}