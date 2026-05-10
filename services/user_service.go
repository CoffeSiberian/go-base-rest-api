package services

import (
	"context"
	"errors"
	"net/http"
	"time"

	"gin-hola-mundo/models"
	"gin-hola-mundo/repositories"
	"gin-hola-mundo/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var validRoles = map[string]bool{
	"admin": true, "teacher": true, "student": true, "parent": true,
}

type CreateUserRequest struct {
	Name     string `json:"name"     binding:"required"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role"     binding:"required"`
}

type UserResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type UserService interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error)
}

type userService struct {
	repo repositories.UserRepository
}

func NewUserService(repo repositories.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	if !validRoles[req.Role] {
		return nil, &utils.AppError{HTTPCode: http.StatusUnprocessableEntity, Code: "INVALID_ROLE", Message: "role must be one of: admin, teacher, student, parent"}
	}

	_, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil {
		return nil, &utils.AppError{HTTPCode: http.StatusConflict, Code: "EMAIL_TAKEN", Message: "email already in use"}
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, utils.ErrInternal
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, utils.ErrInternal
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hash),
		Role:     req.Role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, utils.ErrInternal
	}

	return &UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}
