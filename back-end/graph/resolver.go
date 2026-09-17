package graph

import (
	"bookmyvenue/internal/service"

	"gorm.io/gorm"
)

type Resolver struct {
	DB            *gorm.DB
	AdminVenueSvc service.AdminVenueService
}
