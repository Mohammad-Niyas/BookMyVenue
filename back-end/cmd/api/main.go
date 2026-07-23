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

	// 1. Repositories
	userRepo    := repository.NewUserRepository(db)
	adminRepo   := repository.NewAdminRepository(db)
	venueRepo   := repository.NewVenueRepository(db)
	spaceRepo   := repository.NewSpaceRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	// 2. Services (Now spaceRepo, venueRepo, and rdb exist above!)
	authService       := service.NewAuthService(userRepo, cfg)
	adminAuthService  := service.NewAdminAuthService(adminRepo, cfg)
	venueService      := service.NewVenueService(venueRepo, spaceRepo, s3Client, rdb)
	adminVenueService := service.NewAdminVenueService(venueRepo)
	bookingService    := service.NewBookingService(bookingRepo, spaceRepo, venueRepo, rdb)
	// 3. Handlers
	authHandler       := handler.NewAuthHandler(authService)
	adminAuthHandler  := handler.NewAdminAuthHandler(adminAuthService)
	venueHandler      := handler.NewVenueHandler(venueService)
	adminVenueHandler := handler.NewAdminVenueHandler(adminVenueService)
	bookingHandler    := handler.NewBookingHandler(bookingService)

	r := router.SetupRouter(cfg,rdb, authHandler,adminAuthHandler,venueHandler,adminVenueHandler,bookingHandler)

	port := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("🚀 BookMyVenue server starting on port %s", cfg.ServerPort)
	if err := r.Run(port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}