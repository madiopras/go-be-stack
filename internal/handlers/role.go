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

func GetRoles(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(
		"SELECT id, name, code, description, created_at FROM roles ORDER BY id")
	if err != nil {
		log.Printf("Error fetching roles: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching roles")
		return
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.CreatedAt)
		if err != nil {
			log.Printf("Error scanning role: %v", err)
			SendError(w, http.StatusInternalServerError, "Error scanning role")
			return
		}
		roles = append(roles, role)
	}

	SendSuccess(w, http.StatusOK, "Roles retrieved successfully", roles)
}

func GetRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var role models.Role
	err = database.DB.QueryRow(
		"SELECT id, name, code, description, created_at FROM roles WHERE id=$1", id,
	).Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.CreatedAt)
	if err != nil {
		SendError(w, http.StatusNotFound, "Role not found")
		return
	}

	SendSuccess(w, http.StatusOK, "Role retrieved successfully", role)
}

func CreateRole(w http.ResponseWriter, r *http.Request) {
	var role models.Role
	err := json.NewDecoder(r.Body).Decode(&role)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if role.Name == "" || role.Code == "" {
		SendError(w, http.StatusBadRequest, "name and code are required")
		return
	}

	err = database.DB.QueryRow(
		"INSERT INTO roles (name, code, description) VALUES ($1, $2, $3) RETURNING id, created_at",
		role.Name, role.Code, role.Description,
	).Scan(&role.ID, &role.CreatedAt)
	if err != nil {
		log.Printf("Error creating role: %v", err)
		SendError(w, http.StatusInternalServerError, "Error creating role")
		return
	}

	SendSuccess(w, http.StatusCreated, "Role created successfully", role)
}

func UpdateRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var role models.Role
	err = json.NewDecoder(r.Body).Decode(&role)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := database.DB.Exec(
		"UPDATE roles SET name=$1, code=$2, description=$3 WHERE id=$4",
		role.Name, role.Code, role.Description, id)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error updating role")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		SendError(w, http.StatusNotFound, "Role not found")
		return
	}

	role.ID = id
	SendSuccess(w, http.StatusOK, "Role updated successfully", role)
}

func DeleteRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	result, err := database.DB.Exec("DELETE FROM roles WHERE id=$1", id)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error deleting role")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		SendError(w, http.StatusNotFound, "Role not found")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "Role deleted successfully")
}
