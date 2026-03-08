package service

import (
	"errors"
	"time"
)

var (
	ErrEmailTaken            = errors.New("email address is already registered")
	ErrOrganizationNameTaken = errors.New("organization name is already taken")
	ErrInvalidOTP            = errors.New("invalid or expired verification code")
	ErrOTPExpired            = errors.New("verification code has expired")
	ErrOTPLocked             = errors.New("too many incorrect attempts, account locked")
	ErrAlreadyVerified       = errors.New("email is already verified")
	ErrInvalidIndustry       = errors.New("invalid industry value")
	ErrInvalidOrgSize        = errors.New("invalid organization size")
	ErrInvitationNotFound    = errors.New("invitation not found")
	ErrInvitationExpired     = errors.New("invitation has expired")
	ErrInvitationInvalid     = errors.New("invalid invitation token")
	ErrInvitationAlreadySent = errors.New("pending invitation already exists for this email")
)

func otpExpiry() time.Time {
	return time.Now().Add(10 * time.Minute)
}

func isExpired(t time.Time) bool {
	return time.Now().After(t)
}
