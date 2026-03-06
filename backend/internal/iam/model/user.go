package model

import "time"

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	AvatarURL    *string
	Status       UserStatus
	MFAEnabled   bool
	MFASecret    *string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedAt    *time.Time

	// Populated via joins
	Roles []Role
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

func (u *User) RoleSlugs() []string {
	slugs := make([]string, len(u.Roles))
	for i, r := range u.Roles {
		slugs[i] = r.Slug
	}
	return slugs
}

func (u *User) AllPermissions() []string {
	seen := make(map[string]struct{})
	var perms []string
	for _, r := range u.Roles {
		for _, p := range r.Permissions {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				perms = append(perms, p)
			}
		}
	}
	return perms
}
