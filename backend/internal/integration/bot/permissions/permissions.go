package permissions

import (
	"strings"

	"github.com/clario360/platform/internal/auth"
	iamdto "github.com/clario360/platform/internal/iam/dto"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
)

func RequireLinkedUser(cmd bottypes.BotCommand) error {
	if cmd.User == nil {
		return bottypes.ErrUnlinkedUser
	}
	return nil
}

func UserHasPermission(user *iamdto.UserResponse, required string) bool {
	if user == nil {
		return false
	}

	roleSlugs := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleSlugs = append(roleSlugs, role.Slug)
		for _, perm := range role.Permissions {
			if perm == required || perm == "*:*" {
				return true
			}
			if strings.HasSuffix(perm, ":*") {
				prefix := strings.TrimSuffix(perm, "*")
				if strings.HasPrefix(required, prefix) {
					return true
				}
			}
		}
	}
	return auth.HasPermission(roleSlugs, required)
}
