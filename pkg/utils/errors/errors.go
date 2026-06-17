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
	ErrUnsupportedGitProvider       = stderrors.New("unsupported git provider")
	ErrInvalidGitAccessToken        = stderrors.New("invalid git access token")
	ErrFailedToSaveGitIntegration   = stderrors.New("failed to save git integration")
	ErrInvalidBaseURL               = stderrors.New("invalid base url")
	ErrFailedToFetchGitProjects     = stderrors.New("failed to fetch git projects")
	ErrFailedToSaveGitProjects      = stderrors.New("failed to save git projects")
)
