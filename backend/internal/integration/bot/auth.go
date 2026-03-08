package bot

import (
	iamdto "github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/integration/bot/permissions"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
)

func RequireLinkedUser(cmd bottypes.BotCommand) error {
	return permissions.RequireLinkedUser(cmd)
}

func UserHasPermission(user *iamdto.UserResponse, required string) bool {
	return permissions.UserHasPermission(user, required)
}
