package handlers

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateOrderRequest struct {
	UserID int64   `json:"user_id" binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
}
