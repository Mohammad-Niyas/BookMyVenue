package service

import (
	"bookmyvenue/config"
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"bookmyvenue/pkg/utils"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AuthService interface {
	RegisterUser(req RegisterRequest) (*AuthResponse, error)
	RegisterOwner(req OwnerRegisterRequest) (*AuthResponse, error)
	Login(req LoginRequest) (*AuthResponse, error)
	RefreshToken(req RefreshTokenRequest) (*AuthResponse, error)
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
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Role         string `json:"role"`
}
type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
	rdb      *redis.Client
}
func NewAuthService(userRepo repository.UserRepository, cfg *config.Config, rdb *redis.Client) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
		rdb:      rdb,
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

	// Persist refresh token in Redis
	ctx := context.Background()
	_ = s.storeRefreshToken(ctx, user.ID.String(), tokenPair.RefreshToken)

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

	// Persist refresh token in Redis
	ctx := context.Background()
	_ = s.storeRefreshToken(ctx, user.ID.String(), tokenPair.RefreshToken)

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

	// Persist refresh token in Redis
	ctx := context.Background()
	_ = s.storeRefreshToken(ctx, user.ID.String(), tokenPair.RefreshToken)

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		Role:         user.Role,
	}, nil
}

// RefreshToken validates the token in Redis, burns the old token, and issues a rotated pair
func (s *authService) RefreshToken(req RefreshTokenRequest) (*AuthResponse, error) {
	// 1. Verify cryptographic JWT signature
	claims, err := utils.ValidateToken(req.RefreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}
	ctx := context.Background()
	tokenHash := utils.HashToken(req.RefreshToken)
	redisKey := fmt.Sprintf("refresh:%s", tokenHash)
	// 2. Check if token exists in Redis whitelist
	storedUserID, err := s.rdb.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, errors.New("refresh token revoked or expired")
	}
	// 3. Verify user matches and is still active in database
	if storedUserID != claims.UserID.String() {
		return nil, errors.New("token user mismatch")
	}
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user.Status != "active" {
		return nil, errors.New("account is " + user.Status)
	}
	// 4. Generate new token pair
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
	// 5. REFRESH TOKEN ROTATION (RTR):
	// Delete the old token from Redis immediately (burned)
	_ = s.rdb.Del(ctx, redisKey)
	// Store the new token in Redis
	_ = s.storeRefreshToken(ctx, user.ID.String(), tokenPair.RefreshToken)
	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		Role:         user.Role,
	}, nil
}

  

// Helper to store refresh token hash in Redis
func (s *authService) storeRefreshToken(ctx context.Context, userID string, refreshToken string) error {
	tokenHash := utils.HashToken(refreshToken)
	ttl := time.Duration(s.cfg.RefreshTokenExpiryDays) * 24 * time.Hour
	return s.rdb.Set(ctx, fmt.Sprintf("refresh:%s", tokenHash), userID, ttl).Err()
}