package types

import (
	"errors"

	iamdto "github.com/clario360/platform/internal/iam/dto"
)

type BotCommand struct {
	Subcommand string
	Args       []string
	User       *iamdto.UserResponse
	Token      string
	TenantID   string
	Platform   string
	RawText    string
}

type BotResponse struct {
	Text      string
	DataType  string
	Data      any
	Ephemeral bool
	InThread  bool
}

var (
	ErrUnlinkedUser    = errors.New("your account is not linked to Clario 360")
	ErrPermissionDenied = errors.New("you do not have permission to execute this command")
)
