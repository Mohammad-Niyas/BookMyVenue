package repository

import (
	"bookmyvenue/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SpaceRepository interface {
	Create(space *domain.Space) error
	FindByID(id uuid.UUID) (*domain.Space, error)
	FindByVenueID(venueID uuid.UUID) ([]domain.Space, error)
	Update(space *domain.Space) error
	Delete(id uuid.UUID) error
}
type spaceRepository struct {
	db *gorm.DB
}

func NewSpaceRepository(db *gorm.DB) SpaceRepository {
	return &spaceRepository{db: db}
}
func (r *spaceRepository) Create(space *domain.Space) error {
	return r.db.Create(space).Error
}
func (r *spaceRepository) FindByID(id uuid.UUID) (*domain.Space, error) {
	var space domain.Space
	err := r.db.First(&space, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &space, nil
}
func (r *spaceRepository) FindByVenueID(venueID uuid.UUID) ([]domain.Space, error) {
	var spaces []domain.Space
	err := r.db.Where("venue_id = ?", venueID).
		Order("created_at ASC").
		Find(&spaces).Error
	if err != nil {
		return nil, err
	}
	return spaces, nil
}
func (r *spaceRepository) Update(space *domain.Space) error {
	return r.db.Save(space).Error
}
func (r *spaceRepository) Delete(id uuid.UUID) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        if err := tx.Where("space_id = ?", id).Delete(&domain.Slot{}).Error; err != nil {
            return err
        }
        return tx.Delete(&domain.Space{}, "id = ?", id).Error
    })
}