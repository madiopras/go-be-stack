package handlers

import (
	"betest/internal/database"
	"betest/internal/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func GetPermissions(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(
		"SELECT id, name, code, resource, action, description, created_at FROM permissions ORDER BY resource, action")
	if err != nil {
		log.Printf("Error fetching permissions: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching permissions")
		return
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var p models.Permission
		err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.Resource, &p.Action, &p.Description, &p.CreatedAt)
		if err != nil {
			log.Printf("Error scanning permission: %v", err)
			SendError(w, http.StatusInternalServerError, "Error scanning permission")
			return
		}
		permissions = append(permissions, p)
	}

	SendSuccess(w, http.StatusOK, "Permissions retrieved successfully", permissions)
}

func GetPermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var p models.Permission
	err = database.DB.QueryRow(
		"SELECT id, name, code, resource, action, description, created_at FROM permissions WHERE id=$1", id,
	).Scan(&p.ID, &p.Name, &p.Code, &p.Resource, &p.Action, &p.Description, &p.CreatedAt)
	if err != nil {
		SendError(w, http.StatusNotFound, "Permission not found")
		return
	}

	SendSuccess(w, http.StatusOK, "Permission retrieved successfully", p)
}

func GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roleID, err := strconv.Atoi(vars["roleId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	rows, err := database.DB.Query(
		`SELECT p.id, p.name, p.code, p.resource, p.action, p.description, p.created_at
		 FROM permissions p
		 INNER JOIN role_permissions rp ON rp.permission_id = p.id
		 WHERE rp.role_id = $1 ORDER BY p.resource, p.action`, roleID)
	if err != nil {
		log.Printf("Error fetching role permissions: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching role permissions")
		return
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var p models.Permission
		err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.Resource, &p.Action, &p.Description, &p.CreatedAt)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "Error scanning permission")
			return
		}
		permissions = append(permissions, p)
	}

	SendSuccess(w, http.StatusOK, "Role permissions retrieved successfully", permissions)
}

func AssignPermissionToRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roleID, err := strconv.Atoi(vars["roleId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}
	permissionID, err := strconv.Atoi(vars["permissionId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid permission ID")
		return
	}

	_, err = database.DB.Exec(
		"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT (role_id, permission_id) DO NOTHING",
		roleID, permissionID)
	if err != nil {
		log.Printf("Error assigning permission to role: %v", err)
		SendError(w, http.StatusInternalServerError, "Error assigning permission")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "Permission assigned to role successfully")
}

func RevokePermissionFromRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roleID, err := strconv.Atoi(vars["roleId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}
	permissionID, err := strconv.Atoi(vars["permissionId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid permission ID")
		return
	}

	result, err := database.DB.Exec(
		"DELETE FROM role_permissions WHERE role_id=$1 AND permission_id=$2", roleID, permissionID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error revoking permission")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		SendError(w, http.StatusNotFound, "Role permission not found")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "Permission revoked from role successfully")
}
