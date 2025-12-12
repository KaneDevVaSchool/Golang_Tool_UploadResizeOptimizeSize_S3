package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// APIResponse đại diện cho standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError đại diện cho API error structure
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BaseHandler cung cấp common handler functionality
type BaseHandler struct{}

// SendSuccess gửi successful JSON response
func (b *BaseHandler) SendSuccess(w http.ResponseWriter, data interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	b.SendJSON(w, http.StatusOK, response)
}

// SendError gửi error JSON response
func (b *BaseHandler) SendError(w http.ResponseWriter, statusCode int, code, message string) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
	b.SendJSON(w, statusCode, response)
}

// SendJSON gửi JSON response
func (b *BaseHandler) SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[BaseHandler] Error encoding JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
