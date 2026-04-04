package model

import "errors"

var (
	ErrExpiredInvitation             = errors.New("invitation expired")
	ErrInvitationUsed                = errors.New("invitation already used")
	ErrDuplicatePendingInvitation    = errors.New("pending invitation already exists for this email address")
)
