package response

import (
	"encoding/json"
	"net/http"
)

// PaginationMeta adalah metadata untuk pagination
type PaginationMeta struct {
	Page       int `json:"page"`        // Halaman saat ini
	PerPage    int `json:"per_page"`   // Jumlah data per halaman
	Total      int `json:"total"`       // Total semua data
	TotalPages int `json:"total_pages"` // Total halaman
}

// PaginatedResponse adalah struktur untuk respons dengan pagination
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SendPaginatedSuccess mengirim respons sukses dengan pagination metadata
func SendPaginatedSuccess(w http.ResponseWriter, statusCode int, message string, data interface{}, meta PaginationMeta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(PaginatedResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}
