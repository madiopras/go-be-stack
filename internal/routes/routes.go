package routes

import (
	"betest/internal/handlers"
	"betest/internal/middleware"

	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Auth routes (public)
	r.HandleFunc("/register", handlers.Register).Methods("POST")
	r.HandleFunc("/login", handlers.Login).Methods("POST")
	r.HandleFunc("/refresh", handlers.RefreshToken).Methods("POST")

	// Protected routes (require Authorization: Bearer <access_token>)
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(middleware.JWTMiddleware)

	protected.HandleFunc("/logout", handlers.Logout).Methods("POST")
	protected.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	protected.HandleFunc("/users/{id}", handlers.GetUser).Methods("GET")
	protected.HandleFunc("/users", handlers.CreateUser).Methods("POST")
	protected.HandleFunc("/users/{id}", handlers.UpdateUser).Methods("PUT")
	protected.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")

	return r
}
