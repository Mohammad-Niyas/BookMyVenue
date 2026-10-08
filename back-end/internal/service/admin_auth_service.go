package service

import (
	"bookmyvenue/config"
	"bookmyvenue/internal/repository"
	"bookmyvenue/pkg/utils"
	"errors"

	"gorm.io/gorm"
)

type AdminAuthService interface {
	Login(req LoginRequest) (*AuthResponse, error)
}

type adminAuthService struct {
	adminRepo repository.AdminRepository
	cfg       *config.Config
}

func NewAdminAuthService(adminRepo repository.AdminRepository, cfg *config.Config) AdminAuthService {
	return &adminAuthService{
		adminRepo: adminRepo,
		cfg:       cfg,
	}
}

func (s *adminAuthService) Login(req LoginRequest) (*AuthResponse, error) {
	admin, err := s.adminRepo.FindByEmail(req.Email)
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, errors.New("internal server error")
	}

	if !utils.CheckPasswordHash(req.Password, admin.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	accessToken, err := utils.GenerateAccessToken(
		admin.ID,
		"admin",
		s.cfg.JWTSecret,
		s.cfg.AccessTokenExpiryMins,
	)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: "", // Disabled for Admin
		Role:         "admin",
	}, nil
}