package handlers

import (
	"betest/internal/database"
	"betest/internal/models"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Rdb        *redis.Client // Exported untuk digunakan di middleware
)

func init() {
	// Generate RSA keys for demo (in production, load from files)
	var err error
	PrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	PublicKey = &PrivateKey.PublicKey

	// Redis client
	Rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Update if needed
	})
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User         models.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}

	// Insert user
	var user models.User
	err = database.DB.QueryRow("INSERT INTO users (name, email, password) VALUES ($1, $2, $3) RETURNING id, name, email, created_at",
		req.Name, req.Email, string(hashedPassword)).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := generateTokens(user.ID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	// Set refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		Path:     "/",
		MaxAge:   7 * 24 * 3600, // 7 days
	})

	SendSuccess(w, http.StatusCreated, "User registered successfully", AuthResponse{
		User:        user,
		AccessToken: accessToken,
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get user
	var user models.User
	err = database.DB.QueryRow("SELECT id, name, email, password, created_at FROM users WHERE email=$1", req.Email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		SendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		SendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := generateTokens(user.ID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	// Set refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
	})

	SendSuccess(w, http.StatusOK, "Login successful", AuthResponse{
		User:        user,
		AccessToken: accessToken,
	})
}

func RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		SendError(w, http.StatusUnauthorized, "No refresh token")
		return
	}

	// Parse refresh token
	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return PublicKey, nil
	})
	if err != nil || !token.Valid {
		SendError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		SendError(w, http.StatusUnauthorized, "Invalid token claims")
		return
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		SendError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		SendError(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	userID := int(userIDFloat)

	// Check if jti exists in Redis
	val, err := Rdb.Get(context.Background(), fmt.Sprintf("refresh_token:%s", jti)).Result()
	if err != nil || val != fmt.Sprintf("%d", userID) {
		SendError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// Blacklist access token lama yang terkait dengan refresh token ini
	// (agar token lama tidak bisa dipakai setelah dapat token baru)
	oldAccessJti, err := Rdb.Get(context.Background(), fmt.Sprintf("refresh_to_access:%s", jti)).Result()
	if err == nil && oldAccessJti != "" {
		// Blacklist dengan TTL 15 menit (maksimal sisa umur access token)
		Rdb.Set(context.Background(), fmt.Sprintf("blacklist:access_token:%s", oldAccessJti), "1", 15*time.Minute)
	}

	// Delete old refresh token dan mapping (rotation)
	Rdb.Del(context.Background(), fmt.Sprintf("refresh_token:%s", jti))
	Rdb.Del(context.Background(), fmt.Sprintf("refresh_to_access:%s", jti))

	// Generate new tokens
	accessToken, refreshToken, err := generateTokens(userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	// Set new refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
	})

	SendSuccess(w, http.StatusOK, "Token refreshed successfully", map[string]string{"access_token": accessToken})
}

func generateTokens(userID int) (string, string, error) {
	// Access token dengan JTI untuk tracking dan blacklist
	accessJti := uuid.New().String()
	accessClaims := jwt.MapClaims{
		"user_id": userID,
		"jti":     accessJti,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(PrivateKey)
	if err != nil {
		return "", "", err
	}

	// Store access token JTI di Redis dengan expiry 15 menit (sama dengan token expiry)
	// Key format: "access_token:jti" untuk membedakan dari refresh token
	err = Rdb.Set(context.Background(), fmt.Sprintf("access_token:%s", accessJti), fmt.Sprintf("%d", userID), 15*time.Minute).Err()
	if err != nil {
		return "", "", err
	}

	// Refresh token as JWT
	refreshJti := uuid.New().String()
	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"jti":     refreshJti,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(PrivateKey)
	if err != nil {
		return "", "", err
	}

	// Store refresh token jti in Redis with expiry
	err = Rdb.Set(context.Background(), fmt.Sprintf("refresh_token:%s", refreshJti), fmt.Sprintf("%d", userID), 7*24*time.Hour).Err()
	if err != nil {
		return "", "", err
	}

	// Mapping refresh_token JTI -> access_token JTI agar saat refresh bisa blacklist access token lama
	err = Rdb.Set(context.Background(), fmt.Sprintf("refresh_to_access:%s", refreshJti), accessJti, 7*24*time.Hour).Err()
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func Logout(w http.ResponseWriter, r *http.Request) {
	// Parse access token dari Authorization header untuk mendapatkan JTI
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString != "" {
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return PublicKey, nil
			})
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if jti, ok := claims["jti"].(string); ok {
						// Tambahkan access token ke blacklist
						// Hitung sisa waktu expiry token
						if exp, ok := claims["exp"].(float64); ok {
							expTime := time.Unix(int64(exp), 0)
							remainingTime := time.Until(expTime)
							if remainingTime > 0 {
								// Set ke blacklist dengan TTL sesuai sisa waktu token
								Rdb.Set(context.Background(), fmt.Sprintf("blacklist:access_token:%s", jti), "1", remainingTime)
							}
						}
					}
				}
			}
		}
	}

	// Hapus refresh token dari cookie jika ada
	cookie, err := r.Cookie("refresh_token")
	if err == nil {
		// Parse refresh token to get jti
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return PublicKey, nil
		})
		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if jti, ok := claims["jti"].(string); ok {
					// Hapus refresh token dari Redis
					Rdb.Del(context.Background(), fmt.Sprintf("refresh_token:%s", jti))
				}
			}
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		MaxAge:   -1,
	})

	SendSuccessNoData(w, http.StatusOK, "Logout successful")
}
