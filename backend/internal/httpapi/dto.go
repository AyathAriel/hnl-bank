package httpapi

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=2,max=200"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

type DepositRequest struct {
	AccountNumber string `json:"account_number" validate:"required"`
	Amount        string `json:"amount" validate:"required"`
	Description   string `json:"description" validate:"max=280"`
}

type WithdrawRequest struct {
	AccountNumber string `json:"account_number" validate:"required"`
	Amount        string `json:"amount" validate:"required"`
	Description   string `json:"description" validate:"max=280"`
}

type TransferRequest struct {
	FromAccountNumber string `json:"from_account_number" validate:"required"`
	ToAccountNumber   string `json:"to_account_number" validate:"required"`
	Amount            string `json:"amount" validate:"required"`
	Description       string `json:"description" validate:"max=280"`
}

type ChatRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message" validate:"required,min=1,max=2000"`
}

type ChatResponse struct {
	ConversationID string `json:"conversation_id"`
	Reply          string `json:"reply"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
