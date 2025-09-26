package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aishahsofea/go-ai-gateway/internal/db"
	"github.com/aishahsofea/go-ai-gateway/internal/models"
	"github.com/aishahsofea/go-ai-gateway/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthHandler struct {
	userRepo *db.UserRepository
	logger   *log.Logger
}

type registerUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

type loginResponse struct {
	User  authResponse `json:"user"`
	Token string       `json:"token"`
}

func NewAuthHandler(userRepo *db.UserRepository, logger *log.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req registerUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Printf("ERROR: could not decode register request: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request payload"})
		return
	}

	err = validateEmail(req.Email)
	if err != nil {
		h.logger.Printf("ERROR: invalid email format: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid email format"})
		return
	}

	user := &models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Role:      req.Role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// deal with password
	err = user.Password.Set(req.Password)
	if err != nil {
		h.logger.Printf("ERROR: could not hash password: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	err = h.userRepo.CreateUser(r.Context(), user)
	if err != nil {
		h.logger.Printf("ERROR: could not create user: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	response := authResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"data": response})
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if len(email) == 0 {
		return fmt.Errorf("email cannot be empty")
	}

	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(emailRegex, email)

	if !matched {
		return fmt.Errorf("email is not valid")
	}

	return nil
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		h.logger.Printf("ERROR: could not decode login request: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request payload"})
		return
	}

	user, err := h.userRepo.GetUserByEmail(r.Context(), req.Email)
	if err == sql.ErrNoRows {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if err != nil {
		h.logger.Printf("ERROR: could not find user by email: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	passwordMatch, err := user.Password.Matches(req.Password)
	if err != nil {
		h.logger.Printf("ERROR: could not compare password: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	if !passwordMatch {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid credentials"})
		return
	}

	// generate token
	token, err := h.generateJWT(user)
	if err != nil {
		h.logger.Printf("ERROR: could not generate JWT: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	response := loginResponse{
		User: authResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
		Token: token,
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
}

func (h *AuthHandler) generateJWT(user *models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub":      user.ID,
			"username": user.Email,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
