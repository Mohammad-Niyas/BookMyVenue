package repository

import (
	"bookmyvenue/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SpaceRepository interface {
	Create(space *domain.Space) error
	FindBySpaceID(id uuid.UUID) (*domain.Space, error)
	FindByVenueID(venueID uuid.UUID) ([]domain.Space, error)
	Update(space *domain.Space) error
	Delete(id uuid.UUID) error

	CreateSlots(slots []domain.Slot) error
	FindBySlotID(id uuid.UUID)(*domain.Slot,error)
	FindSlotsBySpaceIDAndDate(spaceID uuid.UUID, date time.Time) ([]domain.Slot, error)
	DeleteUnbookedSlotsByDate(spaceID uuid.UUID, date time.Time) error
	ReplaceSlots(spaceID uuid.UUID, date time.Time, slotsToCreate []domain.Slot) error
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
func (r *spaceRepository) FindBySpaceID(id uuid.UUID) (*domain.Space, error) {
	var space domain.Space
	err := r.db.First(&space, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &space, nil
}
func (r *spaceRepository) FindByVenueID(venueID uuid.UUID) ([]domain.Space, error) {
	var space []domain.Space
	err := r.db.Where("venue_id = ?", venueID).
		Order("created_at ASC").
		Find(&space).Error
	if err != nil {
		return nil, err
	}
	return space, nil
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

// Slot

func (r *spaceRepository) CreateSlots(slots []domain.Slot) error {
	return r.db.Create(&slots).Error
}

func (r *spaceRepository)FindBySlotID(id uuid.UUID)(*domain.Slot,error){
	var slot domain.Slot
	err:=r.db.First(&slot,"id = ?",id).Error
	if err!=nil{
		return nil,err
	}
	return &slot,nil
}

func (r *spaceRepository) FindSlotsBySpaceIDAndDate(spaceID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	var slots []domain.Slot
	err := r.db.Where("space_id = ? AND date = ?", spaceID, date).
		Order("start_time ASC").
		Find(&slots).Error
	return slots, err
}
func (r *spaceRepository) DeleteUnbookedSlotsByDate(spaceID uuid.UUID, date time.Time) error {
	return r.db.Where("space_id = ? AND date = ? AND is_booked = ?", spaceID, date, false).
		Delete(&domain.Slot{}).Error
}
func (r *spaceRepository) ReplaceSlots(spaceID uuid.UUID, date time.Time, slotsToCreate []domain.Slot) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("space_id = ? AND date = ? AND is_booked = ?", spaceID, date, false).
			Delete(&domain.Slot{}).Error; err != nil {
			return err
		}
		if len(slotsToCreate) > 0 {
			if err := tx.Create(&slotsToCreate).Error; err != nil {
				return err
			}
		}
		return nil
	})
}