package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"deliveryService/models"

	"github.com/golang-jwt/jwt"
)

type LoginHandler struct {
	DB        *sql.DB
	JWTSecret string
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginReq models.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	

	var user models.User
	err = h.DB.QueryRow(
		"SELECT id, name, password, role, address FROM users WHERE name = ?",
		loginReq.Name,
	).Scan(&user.ID, &user.Name, &user.Password, &user.Role, &user.Address)

	if err == sql.ErrNoRows {
		http.Error(w, "Usuario no encontrado", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Verificar password (en producción, usar bcrypt)
	if user.Password != loginReq.Password {
		http.Error(w, "Contraseña incorrecta", http.StatusUnauthorized)
		return
	}

	// Generar JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 24 horas
	})

	tokenString, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		http.Error(w, "Error al generar token", http.StatusInternalServerError)
		return
	}

	// No enviar password
	user.Password = ""

	response := models.LoginResponse{
		Token: tokenString,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *LoginHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	
	if user.Role != "customer" && user.Role != "delivery" {
		http.Error(w, "Rol inválido", http.StatusBadRequest)
		return
	}

	
	result, err := h.DB.Exec(
		"INSERT INTO users (name, password, role, address) VALUES (?, ?, ?, ?)",
		user.Name, user.Password, user.Role, user.Address,
	)
	if err != nil {
		http.Error(w, "Error al registrar usuario: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	user.ID = int(id)
	user.Password = ""

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sesión cerrada exitosamente",
	})
}