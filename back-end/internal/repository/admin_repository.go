package repository

import (
	"bookmyvenue/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminRepository interface {
	FindByEmail(email string) (*domain.Admin, error)
	FindByID(id uuid.UUID) (*domain.Admin, error)
}
type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) FindByEmail(email string) (*domain.Admin, error) {
	var admin domain.Admin
	err := r.db.Where("email = ?", email).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) FindByID(id uuid.UUID) (*domain.Admin, error) {
	var admin domain.Admin
	err := r.db.Where("id = ?", id).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}