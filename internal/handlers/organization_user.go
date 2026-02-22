package handlers

import (
	"betest/internal/database"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func GetOrganizationUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	rows, err := database.DB.Query(
		`SELECT ou.id, ou.organization_id, ou.user_id, ou.role_id, ou.created_at,
		  u.name, u.email,
		  r.name, r.code
		  FROM organization_users ou
		  INNER JOIN users u ON u.id = ou.user_id
		  LEFT JOIN roles r ON r.id = ou.role_id
		  WHERE ou.organization_id = $1 ORDER BY ou.created_at DESC`, orgID)
	if err != nil {
		log.Printf("Error fetching organization users: %v", err)
		SendError(w, http.StatusInternalServerError, "Error fetching organization members")
		return
	}
	defer rows.Close()

	type OrgMember struct {
		ID             int     `json:"id"`
		OrganizationID int     `json:"organization_id"`
		UserID         int     `json:"user_id"`
		UserName       string  `json:"user_name"`
		UserEmail      string  `json:"user_email"`
		RoleID         *int    `json:"role_id,omitempty"`
		RoleName       *string `json:"role_name,omitempty"`
		RoleCode       *string `json:"role_code,omitempty"`
		CreatedAt      string  `json:"created_at"`
	}

	var members []OrgMember
	for rows.Next() {
		var m OrgMember
		var roleID sql.NullInt32
		var roleName, roleCode sql.NullString
		var createdAt string
		err := rows.Scan(&m.ID, &m.OrganizationID, &m.UserID, &roleID, &createdAt,
			&m.UserName, &m.UserEmail, &roleName, &roleCode)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "Error scanning member")
			return
		}
		m.CreatedAt = createdAt
		if roleID.Valid {
			id := int(roleID.Int32)
			m.RoleID = &id
		}
		if roleName.Valid {
			m.RoleName = &roleName.String
		}
		if roleCode.Valid {
			m.RoleCode = &roleCode.String
		}
		members = append(members, m)
	}

	SendSuccess(w, http.StatusOK, "Organization members retrieved successfully", members)
}

type AddOrgMemberRequest struct {
	UserID int `json:"user_id"`
	RoleID *int `json:"role_id,omitempty"`
}

func AddOrganizationUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	var req AddOrgMemberRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.UserID == 0 {
		SendError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	var roleVal sql.NullInt32
	if req.RoleID != nil {
		roleVal = sql.NullInt32{Int32: int32(*req.RoleID), Valid: true}
	}
	_, err = database.DB.Exec(
		"INSERT INTO organization_users (organization_id, user_id, role_id) VALUES ($1, $2, $3) ON CONFLICT (organization_id, user_id) DO UPDATE SET role_id = EXCLUDED.role_id",
		orgID, req.UserID, roleVal)
	if err != nil {
		log.Printf("Error adding organization member: %v", err)
		SendError(w, http.StatusInternalServerError, "Error adding member")
		return
	}

	SendSuccessNoData(w, http.StatusCreated, "Member added to organization successfully")
}

func RemoveOrganizationUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := strconv.Atoi(vars["id"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}
	userID, err := strconv.Atoi(vars["userId"])
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	result, err := database.DB.Exec(
		"DELETE FROM organization_users WHERE organization_id=$1 AND user_id=$2", orgID, userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error removing member")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		SendError(w, http.StatusNotFound, "Member not found")
		return
	}

	SendSuccessNoData(w, http.StatusOK, "Member removed from organization successfully")
}
