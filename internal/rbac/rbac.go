package rbac

import (
	"betest/internal/database"
	"context"
)

// GetUserPermissionCodes returns all permission codes for the user (via global user_roles).
func GetUserPermissionCodes(ctx context.Context, userID int) ([]string, error) {
	query := `
		SELECT DISTINCT p.code
		FROM permissions p
		INNER JOIN role_permissions rp ON rp.permission_id = p.id
		INNER JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1
	`
	rows, err := database.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

// HasPermission checks if user has the given permission code.
func HasPermission(ctx context.Context, userID int, permissionCode string) (bool, error) {
	codes, err := GetUserPermissionCodes(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, c := range codes {
		if c == permissionCode {
			return true, nil
		}
	}
	return false, nil
}
