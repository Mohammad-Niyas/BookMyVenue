package service

import (
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RejectVenueRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}
type RejectDraftRequest struct {
	AdminNote string `json:"admin_note" binding:"required,min=5"`
}

type VenueEditDraftResponse struct {
	ID          uuid.UUID `json:"id"`
	VenueID     uuid.UUID `json:"venue_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	Rules       string    `json:"rules"`
	Timings     string    `json:"timings"`
	Images      []string  `json:"images"`
	Status      string    `json:"status"`
	AdminNote   *string   `json:"admin_note"`
}

type AdminVenueService interface {
	GetPendingVenues() ([]VenueResponse, error)
	ApproveVenue(venueID uuid.UUID) error
	RejectVenue(venueID uuid.UUID, reason string) error
	
	GetPendingDrafts() ([]VenueEditDraftResponse, error)
	ApproveEditDraft(draftID uuid.UUID) error
	RejectEditDraft(draftID uuid.UUID, adminNote string) error
}

type adminVenueService struct {
	venueRepo repository.VenueRepository
}
func NewAdminVenueService(venueRepo repository.VenueRepository) AdminVenueService {
	return &adminVenueService{venueRepo: venueRepo}
}

func mapToDraftResponse(d domain.VenueEditDraft) VenueEditDraftResponse {
	return VenueEditDraftResponse{
		ID:          d.ID,
		VenueID:     d.VenueID,
		Name:        d.Name,
		Description: d.Description,
		Type:        d.Type,
		Address:     d.Address,
		City:        d.City,
		Rules:       d.Rules,
		Timings:     d.Timings,
		Images:      d.Images,
		Status:      d.Status,
		AdminNote:   d.AdminNote,
	}
}
func mapToDraftResponses(drafts []domain.VenueEditDraft) []VenueEditDraftResponse {
	responses := make([]VenueEditDraftResponse, len(drafts))
	for i, d := range drafts {
		responses[i] = mapToDraftResponse(d)
	}
	return responses
}

func (s *adminVenueService) GetPendingVenues() ([]VenueResponse, error) {
	venues, err := s.venueRepo.FindPendingVenues()
	if err != nil {
		return nil, errors.New("failed to fetch pending venues")
	}
	return mapToVenueResponses(venues), nil
}

func (s *adminVenueService) ApproveVenue(venueID uuid.UUID) error {
	venue, err := s.venueRepo.FindByID(venueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("venue not found")
		}
		return errors.New("failed to fetch venue")
	}
	if venue.Status != "pending" {
		return errors.New("venue is not in pending status")
	}
	venue.Status = "approved"
	if err := s.venueRepo.Update(venue); err != nil {
		return errors.New("failed to approve venue")
	}
	return nil
}

func (s *adminVenueService) RejectVenue(venueID uuid.UUID, reason string) error {
	venue, err := s.venueRepo.FindByID(venueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("venue not found")
		}
		return errors.New("failed to fetch venue")
	}
	if venue.Status != "pending" {
		return errors.New("venue is not in pending status")
	}
	venue.Status = "rejected"
	venue.RejectionReason = &reason
	if err := s.venueRepo.Update(venue); err != nil {
		return errors.New("failed to reject venue")
	}
	return nil
}

func (s *adminVenueService) GetPendingDrafts() ([]VenueEditDraftResponse, error) {
	drafts, err := s.venueRepo.FindPendingEditDrafts()
	if err != nil {
		return nil, errors.New("failed to fetch pending drafts")
	}
	return mapToDraftResponses(drafts), nil
}

func (s *adminVenueService) ApproveEditDraft(draftID uuid.UUID) error {
	draft, err := s.venueRepo.FindEditDraftByID(draftID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("edit draft not found")
		}
		return errors.New("failed to fetch draft")
	}
	if draft.Status != "pending_review" {
		return errors.New("draft is not in pending status")
	}
	venue, err := s.venueRepo.FindByID(draft.VenueID)
	if err != nil {
		return errors.New("live venue associated with draft not found")
	}

	venue.Name = draft.Name
	venue.Description = draft.Description
	venue.Type = draft.Type
	venue.Address = draft.Address
	venue.City = draft.City
	venue.Rules = draft.Rules
	venue.Timings = draft.Timings
	venue.Images = draft.Images
	draft.Status = "approved"

	if err := s.venueRepo.Update(venue); err != nil {
		return errors.New("failed to merge draft updates to live venue")
	}
	if err := s.venueRepo.UpdateEditDraft(draft); err != nil {
		return errors.New("failed to update draft status")
	}
	return nil
}

func (s *adminVenueService) RejectEditDraft(draftID uuid.UUID, adminNote string) error {
	draft, err := s.venueRepo.FindEditDraftByID(draftID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("edit draft not found")
		}
		return errors.New("failed to fetch draft")
	}
	if draft.Status != "pending_review" {
		return errors.New("draft is not in pending status")
	}
	draft.Status = "rejected"
	draft.AdminNote = &adminNote
	if err := s.venueRepo.UpdateEditDraft(draft); err != nil {
		return errors.New("failed to reject edit draft")
	}
	return nil
}




