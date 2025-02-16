package handlers

import (
	"encoding/json"
	"loto/db"
	"net/http"
)

type PaymentRequest struct {
	CardNumber  string  `json:"cardNumber"`
	ExpiryDate  string  `json:"expiryDate"`
	CVV         string  `json:"cvv"`
	TopupAmount float64 `json:"topupAmount"`
	Username    string  `json:"username"`
}

func ProcessPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"success": false, "error": "Invalid request method"}`, http.StatusMethodNotAllowed)
		return
	}

	var paymentData PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&paymentData); err != nil {
		http.Error(w, `{"success": false, "error": "Invalid input"}`, http.StatusBadRequest)
		return
	}

	// Проверяем, что пользователь существует
	var currentBalance float64
	err := db.DB.QueryRow("SELECT balance FROM users WHERE username = $1", paymentData.Username).Scan(&currentBalance)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	// Обновляем баланс пользователя
	newBalance := currentBalance + paymentData.TopupAmount
	_, err = db.DB.Exec("UPDATE users SET balance = $1 WHERE username = $2", newBalance, paymentData.Username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to update balance",
		})
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"newBalance": newBalance,
	})
}
