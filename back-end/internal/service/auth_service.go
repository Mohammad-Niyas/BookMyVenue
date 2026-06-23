package service

import (
	"bookmyvenue/config"
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"bookmyvenue/pkg/utils"
	"errors"

	"gorm.io/gorm"
)

type AuthService interface {
	RegisterUser(req RegisterRequest) (*AuthResponse, error)
	RegisterOwner(req OwnerRegisterRequest) (*AuthResponse, error)
	Login(req LoginRequest) (*AuthResponse, error)
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type OwnerRegisterRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Phone        string `json:"phone"`
	BusinessName string `json:"business_name"`
	GSTNumber    string `json:"gst_number"`
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Role         string `json:"role"`
}

type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *authService) RegisterUser(req RegisterRequest) (*AuthResponse, error) {
	existingUser, err := s.userRepo.FindByEmail(req.Email)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("internal server error")
	}
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	user := &domain.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "user",
		Status:       "active",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("failed to create user")
	}

	tokenPair, err := utils.GenerateTokenPair(
		user.ID,
		user.Role,
		s.cfg.JWTSecret,
		s.cfg.AccessTokenExpiryMins,
		s.cfg.RefreshTokenExpiryDays,
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		Role:         user.Role,
	}, nil
}

func (s *authService) RegisterOwner(req OwnerRegisterRequest) (*AuthResponse, error) {
	existingUser, err := s.userRepo.FindByEmail(req.Email)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("internal server error")
	}
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	user := &domain.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "owner",
		Status:       "active",
		Phone:        &req.Phone,
		BusinessName: &req.BusinessName,
		GSTNumber:    &req.GSTNumber,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("failed to create owner")
	}

	tokenPair, err := utils.GenerateTokenPair(
		user.ID,
		user.Role,
		s.cfg.JWTSecret,
		s.cfg.AccessTokenExpiryMins,
		s.cfg.RefreshTokenExpiryDays,
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		Role:         user.Role,
	}, nil
}

func (s *authService) Login(req LoginRequest) (*AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, errors.New("internal server error")
	}

	if user.Status != "active" {
		return nil, errors.New("account is " + user.Status)
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}
	
	tokenPair, err := utils.GenerateTokenPair(
		user.ID,
		user.Role,
		s.cfg.JWTSecret,
		s.cfg.AccessTokenExpiryMins,
		s.cfg.RefreshTokenExpiryDays,
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}
	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		Role:         user.Role,
	}, nil
}