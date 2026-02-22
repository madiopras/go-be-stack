package models

import "time"

type Organization struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Code        string    `json:"code" db:"code"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type OrganizationUser struct {
	ID             int        `json:"id" db:"id"`
	OrganizationID int        `json:"organization_id" db:"organization_id"`
	UserID         int        `json:"user_id" db:"user_id"`
	RoleID         *int       `json:"role_id,omitempty" db:"role_id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}
