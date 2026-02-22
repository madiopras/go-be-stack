package handlers

import (
	"betest/internal/database"
	"betest/internal/models"
	"betest/internal/response"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func GetOrganizations(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 10
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				limit = 100
			} else {
				limit = l
			}
		}
	}
	offset := (page - 1) * limit

	var total int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM organizations").Scan(&total)
	if err != nil {
		log.Printf("Error counting organizations: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching organizations count")
		return
	}

	rows, err := database.DB.Query(
		"SELECT id, name, code, description, created_at FROM organizations ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		log.Printf("Error fetching organizations: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching organizations")
		return
	}
	defer rows.Close()

	var orgs []models.Organization
	for rows.Next() {
		var o models.Organization
		err := rows.Scan(&o.ID, &o.Name, &o.Code, &o.Description, &o.CreatedAt)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "Error scanning organization")
			return
		}
		orgs = append(orgs, o)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	meta := response.PaginationMeta{
		Page: page, PerPage: limit, Total: total, TotalPages: totalPages,
	}
	response.SendPaginatedSuccess(w, http.StatusOK, "Organizations retrieved successfully", orgs, meta)
}

func GetOrganization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var o models.Organization
	err = database.DB.QueryRow(
		"SELECT id, name, code, description, created_at FROM organizations WHERE id=$1", id,
	).Scan(&o.ID, &o.Name, &o.Code, &o.Description, &o.CreatedAt)
	if err != nil {
		SendError(w, http.StatusNotFound, "Organization not found")
		return
	}

	SendSuccess(w, http.StatusOK, "Organization retrieved successfully", o)
}

func CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var o models.Organization
	err := json.NewDecoder(r.Body).Decode(&o)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if o.Name == "" || o.Code == "" {
		SendError(w, http.StatusBadRequest, "name and code are required")
		return
	}

	err = database.DB.QueryRow(
		"INSERT INTO organizations (name, code, description) VALUES ($1, $2, $3) RETURNING id, created_at",
		o.Name, o.Code, o.Description,
	).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		log.Printf("Error creating organization: %v", err)
		SendError(w, http.StatusInternalServerError, "Error creating organization")
		return
	}

	SendSuccess(w, http.StatusCreated, "Organization created successfully", o)
}

func UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var o models.Organization
	err = json.NewDecoder(r.Body).Decode(&o)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := database.DB.Exec(
		"UPDATE organizations SET name=$1, code=$2, description=$3 WHERE id=$4",
		o.Name, o.Code, o.Description, id)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error updating organization")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		SendError(w, http.StatusNotFound, "Organization not found")
		return
	}

	o.ID = id
	SendSuccess(w, http.StatusOK, "Organization updated successfully", o)
}

func DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	result, err := database.DB.Exec("DELETE FROM organizations WHERE id=$1", id)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error deleting organization")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		SendError(w, http.StatusNotFound, "Organization not found")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "Organization deleted successfully")
}
