package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aishahsofea/go-ai-gateway/internal/db"
	"github.com/aishahsofea/go-ai-gateway/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

type AuthResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

func NewAuthHandler(userRepo *db.UserRepository, logger *log.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Printf("ERROR: could not decode register request: %v", err)
		// TODO: WriteJSON
		return
	}

	err = validateEmail(req.Email)
	if err != nil {
		h.logger.Printf("ERROR: invalid email format: %v", err)
		// TODO: WriteJSON response
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Printf("ERROR: could not hash password: %v", err)
		// TODO: WriteJSON response
		return
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		Role:         req.Role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = h.userRepo.CreateUser(r.Context(), user)
	if err != nil {
		h.logger.Printf("ERROR: could not create user: %v", err)
		// TODO: WriteJSON response
		return
	}

	response := AuthResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
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
