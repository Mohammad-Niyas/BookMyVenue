package service

import (
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"bookmyvenue/pkg/s3"
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateVenueRequest struct {
	Name        string   `json:"name" binding:"required,min=3,max=150"`
	Description string   `json:"description" binding:"required,min=10"`
	Type        string   `json:"type" binding:"required,oneof=banquet_hall sports_turf conference_room party_hall coworking_space"`
	Address     string   `json:"address" binding:"required"`
	City        string   `json:"city" binding:"required"`
	Rules       string   `json:"rules"`
	Timings     string   `json:"timings" binding:"required"` 
	Images      []string `json:"images" binding:"required"`
}
type CreateSpaceRequest struct {
	Name        string   `json:"name" binding:"required,min=3,max=100"`
	Capacity    int      `json:"capacity" binding:"required,gt=0"`
	Price       float64  `json:"price" binding:"required,gt=0"`
	BookingType string   `json:"booking_type" binding:"required,oneof=hourly daily"`
	Images      []string `json:"images"`
}
type UpdateVenueRequest struct {
	Name        *string  `json:"name" binding:"omitempty,min=3,max=150"`
	Description *string  `json:"description" binding:"omitempty,min=10"`
	Type        *string  `json:"type" binding:"omitempty,oneof=banquet_hall sports_turf conference_room party_hall coworking_space"`
	Address     *string  `json:"address"`
	City        *string  `json:"city"`
	Rules       *string  `json:"rules"`
	Timings     *string  `json:"timings"`
	Images      []string `json:"images"`
}
type CancellationPolicyResponse struct {
	ID                   uuid.UUID `json:"id"`
	VenueID              uuid.UUID `json:"venue_id"`
	FullRefundDays       int       `json:"full_refund_days"`
	FullRefundPercent    float64   `json:"full_refund_percent"`
	PartialRefundDays    int       `json:"partial_refund_days"`
	PartialRefundPercent float64   `json:"partial_refund_percent"`
}
type SpaceResponse struct {
	ID          uuid.UUID `json:"id"`
	VenueID     uuid.UUID `json:"venue_id"`
	Name        string    `json:"name"`
	Capacity    int       `json:"capacity"`
	Price       float64   `json:"price"`
	BookingType string    `json:"booking_type"`
	Images      []string  `json:"images"`
}
type VenueResponse struct {
	ID                 uuid.UUID                   `json:"id"`
	OwnerID            uuid.UUID                   `json:"owner_id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	Type               string                      `json:"type"`
	Address            string                      `json:"address"`
	City               string                      `json:"city"`
	Rules              string                      `json:"rules"`
	Timings            string                      `json:"timings"`
	Status             string                      `json:"status"`
	Images             []string                    `json:"images"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
	Spaces             []SpaceResponse             `json:"spaces,omitempty"`
	CancellationPolicy *CancellationPolicyResponse `json:"cancellation_policy,omitempty"`
}
type PresignedURLResponse struct {
	UploadURL   string `json:"upload_url"`
	DownloadURL string `json:"download_url"`
}

type VenueService interface {
	// Venue
	CreateVenue(ownerID uuid.UUID, req CreateVenueRequest) (*VenueResponse, error)
	GetVenueByID(ownerID uuid.UUID, venueID uuid.UUID) (*VenueResponse, error)
	GetOwnerVenues(ownerID uuid.UUID) ([]VenueResponse, error)
	UpdateVenue(ownerID uuid.UUID, venueID uuid.UUID, req UpdateVenueRequest) (venue *VenueResponse, isDraft bool, err error)
	DeleteVenue(ownerID uuid.UUID, venueID uuid.UUID) error
	ToggleVenueStatus(ownerID uuid.UUID, venueID uuid.UUID) (*VenueResponse, error)

	// Space 
	AddSpace(ownerID uuid.UUID, venueID uuid.UUID, req CreateSpaceRequest) (*SpaceResponse, error)
	UpdateSpace(ownerID uuid.UUID, spaceID uuid.UUID, req CreateSpaceRequest) (*SpaceResponse, error)
	
	DeleteSpace(ownerID uuid.UUID, spaceID uuid.UUID) error

	// S3 
	GeneratePresignedURL(ctx context.Context, fileName string, contentType string) (*PresignedURLResponse, error)
}

type venueService struct {
	venueRepo repository.VenueRepository
	spaceRepo repository.SpaceRepository
	s3Client  s3.S3Client
}

func NewVenueService(venueRepo repository.VenueRepository, spaceRepo repository.SpaceRepository, s3Client s3.S3Client) VenueService {
	return &venueService{
		venueRepo: venueRepo,
		spaceRepo: spaceRepo,
		s3Client:  s3Client,
	}
}

func mapToSpaceResponse(s domain.Space) SpaceResponse {
	return SpaceResponse{
		ID:          s.ID,
		VenueID:     s.VenueID,
		Name:        s.Name,
		Capacity:    s.Capacity,
		Price:       s.Price,
		BookingType: s.BookingType,
		Images:      s.Images,
	}
}
func mapToSpaceResponses(spaces []domain.Space) []SpaceResponse {
	responses := make([]SpaceResponse, len(spaces))
	for i, s := range spaces {
		responses[i] = mapToSpaceResponse(s)
	}
	return responses
}
func mapToCancellationPolicyResponse(p *domain.CancellationPolicy) *CancellationPolicyResponse {
	if p == nil {
		return nil
	}
	return &CancellationPolicyResponse{
		ID:                   p.ID,
		VenueID:              p.VenueID,
		FullRefundDays:       p.FullRefundDays,
		FullRefundPercent:    p.FullRefundPercent,
		PartialRefundDays:    p.PartialRefundDays,
		PartialRefundPercent: p.PartialRefundPercent,
	}
}
func mapToVenueResponse(v *domain.Venue) *VenueResponse {
	var spaces []SpaceResponse
	if len(v.Spaces) > 0 {
		spaces = mapToSpaceResponses(v.Spaces)
	}
	return &VenueResponse{
		ID:                 v.ID,
		OwnerID:            v.OwnerID,
		Name:               v.Name,
		Description:        v.Description,
		Type:               v.Type,
		Address:            v.Address,
		City:               v.City,
		Rules:              v.Rules,
		Timings:            v.Timings,
		Status:             v.Status,
		Images:             v.Images,
		CreatedAt:          v.CreatedAt,
		UpdatedAt:          v.UpdatedAt,
		Spaces:             spaces,
		CancellationPolicy: mapToCancellationPolicyResponse(v.CancellationPolicy),
	}
}
func mapToVenueResponses(venues []domain.Venue) []VenueResponse {
	responses := make([]VenueResponse, len(venues))
	for i, v := range venues {
		responses[i] = *mapToVenueResponse(&v)
	}
	return responses
}

// Regex pattern to match HH:MM-HH:MM (e.g., 09:00-22:00)
var timingsRegex = regexp.MustCompile(`^([0-1][0-9]|2[0-3]):[0-5][0-9]-([0-1][0-9]|2[0-3]):[0-5][0-9]$`)

// Venue

func (s *venueService) CreateVenue(ownerID uuid.UUID, req CreateVenueRequest) (*VenueResponse, error) {
	exists, err := s.venueRepo.ExistsByNameAndAddress(ownerID, req.Name, req.Address)
	if err != nil {
		return nil, errors.New("failed to verify duplicate venue status")
	}
	if exists {
		return nil, errors.New("a venue with this name and address already exists")
	}
	imgCount := len(req.Images)
	if imgCount < 4 || imgCount > 10 {
		return nil, errors.New("venue must have between 4 and 10 images")
	}
	if !timingsRegex.MatchString(req.Timings) {
		return nil, errors.New("invalid timings format: must be HH:MM-HH:MM (e.g., 08:00-22:00)")
	}
	venue := &domain.Venue{
		OwnerID:     ownerID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Address:     req.Address,
		City:        req.City,
		Rules:       req.Rules,
		Timings:     req.Timings,
		Images:      req.Images,
		Status:      "pending",
	}
	if err := s.venueRepo.Create(venue); err != nil {
		return nil, errors.New("failed to create venue")
	}
	return mapToVenueResponse(venue), nil
}

func (s *venueService) GetVenueByID(ownerID uuid.UUID, venueID uuid.UUID) (*VenueResponse, error) {
	venue, err := s.venueRepo.FindByID(venueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("venue not found")
		}
		return nil, errors.New("failed to fetch venue")
	}
	if venue.OwnerID != ownerID {
		return nil, errors.New("unauthorized: you don't own this venue")
	}
	return mapToVenueResponse(venue), nil
}

func (s *venueService) GetOwnerVenues(ownerID uuid.UUID) ([]VenueResponse, error) {
	venues, err := s.venueRepo.FindByOwnerID(ownerID)
	if err != nil {
		return nil, errors.New("failed to fetch venues")
	}
	return mapToVenueResponses(venues), nil
}

func (s *venueService) UpdateVenue(ownerID uuid.UUID, venueID uuid.UUID, req UpdateVenueRequest) (*VenueResponse, bool, error) {
	venue, err := s.venueRepo.FindByID(venueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, errors.New("venue not found")
		}
		return nil, false, errors.New("failed to fetch venue")
	}
	if venue.OwnerID != ownerID {
		return nil, false, errors.New("unauthorized: you don't own this venue")
	}
	if venue.Status == "suspended" {
    	return nil, false, errors.New("cannot edit a suspended venue: please contact admin support")
	}
	if req.Timings != nil {
		if !timingsRegex.MatchString(*req.Timings) {
			return nil, false, errors.New("invalid timings format: must be HH:MM-HH:MM")
		}
	}
	if req.Images != nil {
		imgCount := len(req.Images)
		if imgCount < 4 || imgCount > 10 {
			return nil, false, errors.New("venue must have between 4 and 10 images")
		}
	}
	if venue.Status == "approved" {
		existingDraft, err := s.venueRepo.FindPendingDraftByVenueID(venueID)
		if err == nil && existingDraft != nil {
			if req.Name != nil { existingDraft.Name = *req.Name }
			if req.Description != nil { existingDraft.Description = *req.Description }
			if req.Type != nil { existingDraft.Type = *req.Type }
			if req.Address != nil { existingDraft.Address = *req.Address }
			if req.City != nil { existingDraft.City = *req.City }
			if req.Rules != nil { existingDraft.Rules = *req.Rules }
			if req.Timings != nil { existingDraft.Timings = *req.Timings }
			if req.Images != nil { existingDraft.Images = req.Images }
			if err := s.venueRepo.UpdateEditDraft(existingDraft); err != nil {
				return nil, false, errors.New("failed to update pending edit request")
			}
			return mapToVenueResponse(venue), true, nil
		}
		draft := &domain.VenueEditDraft{
			VenueID:     venueID,
			Name:        venue.Name,
			Description: venue.Description,
			Type:        venue.Type,
			Address:     venue.Address,
			City:        venue.City,
			Rules:       venue.Rules,
			Timings:     venue.Timings,
			Images:      venue.Images,
			Status:      "pending_review",
		}
		if req.Name != nil { draft.Name = *req.Name }
		if req.Description != nil { draft.Description = *req.Description }
		if req.Type != nil { draft.Type = *req.Type }
		if req.Address != nil { draft.Address = *req.Address }
		if req.City != nil { draft.City = *req.City }
		if req.Rules != nil { draft.Rules = *req.Rules }
		if req.Timings != nil { draft.Timings = *req.Timings }
		if req.Images != nil { draft.Images = req.Images }
		if err := s.venueRepo.CreateEditDraft(draft); err != nil {
			return nil, false, errors.New("failed to submit edit request")
		}
		return mapToVenueResponse(venue), true, nil
	}
	if req.Name != nil { venue.Name = *req.Name }
	if req.Description != nil { venue.Description = *req.Description }
	if req.Type != nil { venue.Type = *req.Type }
	if req.Address != nil { venue.Address = *req.Address }
	if req.City != nil { venue.City = *req.City }
	if req.Rules != nil { venue.Rules = *req.Rules }
	if req.Timings != nil { venue.Timings = *req.Timings }
	if req.Images != nil { venue.Images = req.Images }
	if venue.Status == "rejected" {
		venue.Status = "pending"
	}
	if err := s.venueRepo.Update(venue); err != nil {
		return nil, false, errors.New("failed to update venue")
	}
	return mapToVenueResponse(venue), false, nil
}

func (s *venueService) DeleteVenue(ownerID uuid.UUID, venueID uuid.UUID) error {
	venue, err := s.venueRepo.FindByID(venueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("venue not found")
		}
		return errors.New("failed to fetch venue")
	}
	if venue.OwnerID != ownerID {
		return errors.New("unauthorized: you don't own this venue")
	}
	if err := s.venueRepo.Delete(venueID); err != nil {
		return errors.New("failed to delete venue")
	}
	return nil
}

func (s *venueService) ToggleVenueStatus(ownerID uuid.UUID, venueID uuid.UUID) (*VenueResponse, error) {
	venue, err := s.venueRepo.FindByID(venueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("venue not found")
		}
		return nil, errors.New("failed to fetch venue")
	}

	if venue.OwnerID != ownerID {
		return nil, errors.New("unauthorized: you don't own this venue")
	}

	if venue.Status == "approved" || venue.Status == "active" {
		venue.Status = "inactive"
	} else if venue.Status == "inactive" {
		venue.Status = "active"
	} else {
		return nil, errors.New("cannot toggle status: venue is pending or suspended")
	}

	if err := s.venueRepo.Update(venue); err != nil {
		return nil, errors.New("failed to toggle venue status")
	}

	return mapToVenueResponse(venue), nil
}

// Space

func (s *venueService) AddSpace(ownerID uuid.UUID, venueID uuid.UUID, req CreateSpaceRequest) (*SpaceResponse, error) {
	venue, err := s.venueRepo.FindByID(venueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("venue not found")
		}
		return nil, errors.New("failed to fetch venue")
	}
	if venue.OwnerID != ownerID {
		return nil, errors.New("unauthorized: you don't own this venue")
	}
	if venue.Status == "suspended" {
		return nil, errors.New("cannot add spaces to a suspended venue: please contact admin support")
	}
	if len(venue.Spaces) >= 10 {
		return nil, errors.New("maximum limit of 10 spaces per venue has been reached")
	}
	imgCount := len(req.Images)
	if imgCount > 0 { 
		if imgCount > 5 {
			return nil, errors.New("a space cannot have more than 5 images")
		}
	}

	space := &domain.Space{
		VenueID:     venueID,
		Name:        req.Name,
		Capacity:    req.Capacity,
		Price:       req.Price,
		BookingType: req.BookingType,
		Images:      req.Images,
	}

	if err := s.spaceRepo.Create(space); err != nil {
		return nil, errors.New("failed to add space")
	}

	response := mapToSpaceResponse(*space)
	return &response, nil
}

func (s *venueService) UpdateSpace(ownerID uuid.UUID, spaceID uuid.UUID, req CreateSpaceRequest) (*SpaceResponse, error) {
	space, err := s.spaceRepo.FindByID(spaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("space not found")
		}
		return nil, errors.New("failed to fetch space")
	}

	venue, err := s.venueRepo.FindByID(space.VenueID)
	if err != nil {
		return nil, errors.New("failed to fetch venue")
	}

	if venue.OwnerID != ownerID {
		return nil, errors.New("unauthorized: you don't own this venue")
	}

	if venue.Status == "suspended" {
		return nil, errors.New("cannot update spaces of a suspended venue")
	}

	if req.Images != nil {
		if len(req.Images) > 5 {
			return nil, errors.New("a space cannot have more than 5 images")
		}
	}

	space.Name = req.Name
	space.Capacity = req.Capacity
	space.Price = req.Price
	if req.BookingType != "" {
		space.BookingType = req.BookingType
	}
	if req.Images != nil {
		space.Images = req.Images
	}

	if err := s.spaceRepo.Update(space); err != nil {
		return nil, errors.New("failed to update space")
	}

	response := mapToSpaceResponse(*space)
	return &response, nil
}

func (s *venueService) DeleteSpace(ownerID uuid.UUID, spaceID uuid.UUID) error {
	space, err := s.spaceRepo.FindByID(spaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("space not found")
		}
		return errors.New("failed to fetch space")
	}

	venue, err := s.venueRepo.FindByID(space.VenueID)
	if err != nil {
		return errors.New("failed to fetch venue")
	}

	if venue.OwnerID != ownerID {
		return errors.New("unauthorized: you don't own this venue")
	}

	if venue.Status == "suspended" {
		return errors.New("cannot delete spaces of a suspended venue")
	}

	if err := s.spaceRepo.Delete(spaceID); err != nil {
		return errors.New("failed to delete space")
	}

	return nil
}

// S3

func (s *venueService) GeneratePresignedURL(ctx context.Context, fileName string, contentType string) (*PresignedURLResponse, error) {
	uploadURL, downloadURL, err := s.s3Client.GeneratePresignedURL(ctx, fileName, contentType)
	if err != nil {
		return nil, errors.New("failed to generate presigned URL")
	}
	return &PresignedURLResponse{
		UploadURL:   uploadURL,
		DownloadURL: downloadURL,
	}, nil
}




