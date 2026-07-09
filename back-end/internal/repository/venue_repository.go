package repository

import (
	"bookmyvenue/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VenueRepository interface {
	Create(venue *domain.Venue) error
	FindByID(id uuid.UUID) (*domain.Venue, error)
	FindByOwnerID(ownerID uuid.UUID) ([]domain.Venue, error)
	ExistsByNameAndAddress(ownerID uuid.UUID, name string, address string) (bool, error)
	Update(venue *domain.Venue) error
	Delete(id uuid.UUID) error

	CreateCancellationPolicy(policy *domain.CancellationPolicy) error
	FindCancellationPolicyByVenueID(venueID uuid.UUID) (*domain.CancellationPolicy, error)
	UpdateCancellationPolicy(policy *domain.CancellationPolicy) error

	CreateEditDraft(draft *domain.VenueEditDraft) error
	FindPendingDraftByVenueID(venueID uuid.UUID) (*domain.VenueEditDraft, error)
	UpdateEditDraft(draft *domain.VenueEditDraft) error
}
type venueRepository struct {
	db *gorm.DB
}

func NewVenueRepository(db *gorm.DB) VenueRepository {
	return &venueRepository{db: db}
}

// venue

func (r *venueRepository) Create(venue *domain.Venue) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        if err := tx.Create(venue).Error; err != nil {
            return err
        }
        policy := &domain.CancellationPolicy{
            VenueID:              venue.ID,
            FullRefundDays:       15,
            FullRefundPercent:    95.00,
            PartialRefundDays:    7,
            PartialRefundPercent: 50.00,
        }
        return tx.Create(policy).Error
    })
}

func (r *venueRepository) FindByID(id uuid.UUID) (*domain.Venue, error) {
	var venue domain.Venue
	err := r.db.Preload("Spaces").Preload("CancellationPolicy").
		First(&venue, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &venue, nil
}

func (r *venueRepository) FindByOwnerID(ownerID uuid.UUID) ([]domain.Venue, error) {
	var venues []domain.Venue
	err := r.db.Preload("Spaces").Preload("CancellationPolicy").
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Find(&venues).Error
	if err != nil {
		return nil, err
	}
	return venues, nil
}

func (r *venueRepository) ExistsByNameAndAddress(ownerID uuid.UUID, name string, address string) (bool, error) {
    var count int64
    err := r.db.Model(&domain.Venue{}).
        Where("owner_id = ? AND name = ? AND address = ?", ownerID, name, address).
        Count(&count).Error
    return count > 0, err
}

func (r *venueRepository) Update(venue *domain.Venue) error {
	return r.db.Save(venue).Error
}
func (r *venueRepository) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("venue_id = ?", id).Delete(&domain.Space{}).Error; err != nil {
			return err
		}
		if err := tx.Where("venue_id = ?", id).Delete(&domain.CancellationPolicy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("venue_id = ?", id).Delete(&domain.VenueEditDraft{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&domain.Venue{}, "id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

// cancellation policy

func (r *venueRepository) CreateCancellationPolicy(policy *domain.CancellationPolicy) error {
	return r.db.Create(policy).Error
}

func (r *venueRepository) FindCancellationPolicyByVenueID(venueID uuid.UUID) (*domain.CancellationPolicy, error) {
	var policy domain.CancellationPolicy
	err := r.db.Where("venue_id = ?", venueID).First(&policy).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *venueRepository) UpdateCancellationPolicy(policy *domain.CancellationPolicy) error {
	return r.db.Save(policy).Error
}

// edit draft

func (r *venueRepository) CreateEditDraft(draft *domain.VenueEditDraft) error {
	return r.db.Create(draft).Error
}

func (r *venueRepository) FindPendingDraftByVenueID(venueID uuid.UUID) (*domain.VenueEditDraft, error) {
	var draft domain.VenueEditDraft
	err := r.db.Where("venue_id = ? AND status = ?", venueID, "pending_review").
		First(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

func (r *venueRepository) UpdateEditDraft(draft *domain.VenueEditDraft) error {
	return r.db.Save(draft).Error
}

