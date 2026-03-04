package application

// LoginRequest defines the expected payload for the login endpoint.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserDTO defines the basic user profile returned on successful authentication.
type UserDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// LoginResponse defines the JWT tokens and user context returned.
type LoginResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	TokenType    string  `json:"token_type"`
	ExpiresIn    int     `json:"expires_in"`
	User         UserDTO `json:"user"`
}
