package routes

import (
	"betest/internal/handlers"
	"betest/internal/middleware"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	r := mux.NewRouter()
	// CORS is applied in main.go around the whole router so OPTIONS preflight is handled

	// Health check (public, no auth)
	r.HandleFunc("/health", handlers.Health).Methods("GET")

	// Auth routes (public)
	r.HandleFunc("/register", handlers.Register).Methods("POST")
	r.HandleFunc("/login", handlers.Login).Methods("POST")
	r.HandleFunc("/refresh", handlers.RefreshToken).Methods("POST")

	// Protected routes (require JWT)
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(middleware.JWTMiddleware)

	protected.HandleFunc("/logout", handlers.Logout).Methods("POST")

	// Users (RBAC)
	protected.Handle("/users", middleware.RequirePermission("users:list")(http.HandlerFunc(handlers.GetUsers))).Methods("GET")
	protected.Handle("/users/{id}", middleware.RequirePermission("users:read")(http.HandlerFunc(handlers.GetUser))).Methods("GET")
	protected.Handle("/users", middleware.RequirePermission("users:create")(http.HandlerFunc(handlers.CreateUser))).Methods("POST")
	protected.Handle("/users/{id}", middleware.RequirePermission("users:update")(http.HandlerFunc(handlers.UpdateUser))).Methods("PUT")
	protected.Handle("/users/{id}", middleware.RequirePermission("users:delete")(http.HandlerFunc(handlers.DeleteUser))).Methods("DELETE")

	// Roles (RBAC: roles:manage)
	roles := protected.PathPrefix("/roles").Subrouter()
	roles.Use(middleware.RequirePermission("roles:manage"))
	roles.HandleFunc("", handlers.GetRoles).Methods("GET")
	roles.HandleFunc("/{id}", handlers.GetRole).Methods("GET")
	roles.HandleFunc("", handlers.CreateRole).Methods("POST")
	roles.HandleFunc("/{id}", handlers.UpdateRole).Methods("PUT")
	roles.HandleFunc("/{id}", handlers.DeleteRole).Methods("DELETE")

	// Permissions (RBAC: permissions:manage for list/get; roles:manage for role-permission assign/revoke)
	perms := protected.PathPrefix("/permissions").Subrouter()
	perms.Use(middleware.RequirePermission("roles:manage"))
	perms.HandleFunc("", handlers.GetPermissions).Methods("GET")
	perms.HandleFunc("/{id}", handlers.GetPermission).Methods("GET")
	perms.HandleFunc("/roles/{roleId}", handlers.GetRolePermissions).Methods("GET")
	perms.HandleFunc("/roles/{roleId}/permissions/{permissionId}", handlers.AssignPermissionToRole).Methods("POST")
	perms.HandleFunc("/roles/{roleId}/permissions/{permissionId}", handlers.RevokePermissionFromRole).Methods("DELETE")

	// User roles (assign/revoke role to user)
	userRoles := protected.PathPrefix("/users/{userId}/roles").Subrouter()
	userRoles.Use(middleware.RequirePermission("roles:manage"))
	userRoles.HandleFunc("", handlers.GetUserRoles).Methods("GET")
	userRoles.HandleFunc("", handlers.AssignRoleToUser).Methods("POST")
	userRoles.HandleFunc("/{roleId}", handlers.RevokeRoleFromUser).Methods("DELETE")

	// Organizations (RBAC: organizations:manage)
	orgs := protected.PathPrefix("/organizations").Subrouter()
	orgs.Use(middleware.RequirePermission("organizations:manage"))
	orgs.HandleFunc("", handlers.GetOrganizations).Methods("GET")
	orgs.HandleFunc("/{id}", handlers.GetOrganization).Methods("GET")
	orgs.HandleFunc("", handlers.CreateOrganization).Methods("POST")
	orgs.HandleFunc("/{id}", handlers.UpdateOrganization).Methods("PUT")
	orgs.HandleFunc("/{id}", handlers.DeleteOrganization).Methods("DELETE")
	orgs.HandleFunc("/{id}/users", handlers.GetOrganizationUsers).Methods("GET")
	orgs.HandleFunc("/{id}/users", handlers.AddOrganizationUser).Methods("POST")
	orgs.HandleFunc("/{id}/users/{userId}", handlers.RemoveOrganizationUser).Methods("DELETE")

	return r
}
