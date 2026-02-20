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

func GetUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters untuk pagination
	page := 1
	limit := 10 // Default 10 data per halaman

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		} else {
			SendError(w, http.StatusBadRequest, "Invalid page parameter. Must be a positive integer")
			return
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			// Maksimal 100 data per halaman untuk menghindari overload
			if parsedLimit > 100 {
				limit = 100
			} else {
				limit = parsedLimit
			}
		} else {
			SendError(w, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer")
			return
		}
	}

	// Hitung offset
	offset := (page - 1) * limit

	// Query untuk mendapatkan total jumlah data
	var total int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		log.Printf("Error counting users: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching users count")
		return
	}

	// Query untuk mendapatkan data dengan pagination (diurutkan berdasarkan created_at DESC - terbaru dulu)
	rows, err := database.DB.Query(
		"SELECT id, name, email, created_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching users")
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
		if err != nil {
			log.Printf("Error scanning user data: %v", err)
			SendError(w, http.StatusInternalServerError, "Error scanning user data")
			return
		}
		users = append(users, u)
	}

	// Hitung total pages
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	// Kirim response dengan pagination metadata
	meta := response.PaginationMeta{
		Page:       page,
		PerPage:    limit,
		Total:      total,
		TotalPages: totalPages,
	}

	response.SendPaginatedSuccess(w, http.StatusOK, "Users retrieved successfully", users, meta)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var u models.User
	err = database.DB.QueryRow("SELECT id, name, email, created_at FROM users WHERE id=$1", id).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	if err != nil {
		log.Printf("Error getting user %d: %v", id, err)
		SendError(w, http.StatusNotFound, "User not found")
		return
	}

	SendSuccess(w, http.StatusOK, "User retrieved successfully", u)
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = database.DB.QueryRow("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id", u.Name, u.Email).Scan(&u.ID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	SendSuccess(w, http.StatusCreated, "User created successfully", u)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var u models.User
	err = json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := database.DB.Exec("UPDATE users SET name=$1, email=$2 WHERE id=$3", u.Name, u.Email, id)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error updating user")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		SendError(w, http.StatusNotFound, "User not found")
		return
	}

	u.ID = id
	SendSuccess(w, http.StatusOK, "User updated successfully", u)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	result, err := database.DB.Exec("DELETE FROM users WHERE id=$1", id)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error deleting user")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		SendError(w, http.StatusNotFound, "User not found")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "User deleted successfully")
}
