package services

import (
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	apperrors "dsr-automation/pkg/utils/errors"
	"dsr-automation/pkg/jwt"
	"dsr-automation/pkg/utils/passwordhashing"
)

type AuthService interface {
	Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(req *dto.LoginRequest) (*dto.LoginResponse, error)
}

type authService struct {
	users  repository.AuthRepository
	hasher passwordhashing.Hasher
	jwt    jwt.Service
}

func NewAuthService(users repository.AuthRepository, hasher passwordhashing.Hasher, jwtSvc jwt.Service) AuthService {
	return &authService{
		users:  users,
		hasher: hasher,
		jwt:    jwtSvc,
	}
}

func (s *authService) Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	existing, err := s.users.GetByEmail(req.Email)
	if err != nil {
		return nil, apperrors.ErrFailedToCheckEmail
	}
	if existing != nil {
		return nil, apperrors.ErrEmailAlreadyRegistered
	}

	hash, err := s.hasher.HashPassword(req.Password)
	if err != nil {
		return nil, apperrors.ErrFailedToHashPassword
	}

	user := &models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
	}

	if err := s.users.Create(user); err != nil {
		return nil, apperrors.ErrFailedToCreateUser
	}

	return &dto.RegisterResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *authService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.users.GetByEmail(req.Email)
	if err != nil {
		return nil, apperrors.ErrFailedToCheckEmail
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if err := s.hasher.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, apperrors.ErrFailedToGenerateAccessToken
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return nil, apperrors.ErrFailedToGenerateRefreshToken
	}

	return &dto.LoginResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
