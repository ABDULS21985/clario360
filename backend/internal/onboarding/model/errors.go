package model

import "errors"

var (
	ErrExpiredInvitation = errors.New("invitation expired")
	ErrInvitationUsed    = errors.New("invitation already used")
)
