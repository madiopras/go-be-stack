package handlers

import (
	"betest/internal/database"
	"betest/internal/models"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func GetUserRoles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["userId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	rows, err := database.DB.Query(
		`SELECT r.id, r.name, r.code, r.description, r.created_at
		 FROM roles r
		 INNER JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = $1 ORDER BY r.id`, userID)
	if err != nil {
		log.Printf("Error fetching user roles: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching user roles")
		return
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.CreatedAt)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "Error scanning role")
			return
		}
		roles = append(roles, role)
	}

	SendSuccess(w, http.StatusOK, "User roles retrieved successfully", roles)
}

type AssignRoleRequest struct {
	RoleID int `json:"role_id"`
}

func AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["userId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req AssignRoleRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.RoleID == 0 {
		SendError(w, http.StatusBadRequest, "role_id is required")
		return
	}

	_, err = database.DB.Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT (user_id, role_id) DO NOTHING",
		userID, req.RoleID)
	if err != nil {
		log.Printf("Error assigning role to user: %v", err)
		SendError(w, http.StatusInternalServerError, "Error assigning role")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "Role assigned to user successfully")
}

func RevokeRoleFromUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["userId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}
	roleID, err := strconv.Atoi(vars["roleId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	result, err := database.DB.Exec(
		"DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2", userID, roleID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error revoking role")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		SendError(w, http.StatusNotFound, "User role not found")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "Role revoked from user successfully")
}
