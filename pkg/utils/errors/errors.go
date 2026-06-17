package errors

import stderrors "errors"

var (
	ErrEmailAlreadyRegistered = stderrors.New("email already registered")
	ErrFailedToCheckEmail     = stderrors.New("failed to check email")
	ErrFailedToHashPassword   = stderrors.New("failed to hash password")
	ErrFailedToCreateUser         = stderrors.New("failed to create user")
	ErrInvalidCredentials         = stderrors.New("invalid email or password")
	ErrFailedToGenerateAccessToken  = stderrors.New("failed to generate access token")
	ErrFailedToGenerateRefreshToken = stderrors.New("failed to generate refresh token")
)
