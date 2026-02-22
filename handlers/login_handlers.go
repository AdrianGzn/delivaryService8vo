package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"deliveryService/models"
)

type LoginHandler struct {
	DB *sql.DB
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

	// Verificar password (plain text - solo para pruebas)
	if user.Password != loginReq.Password {
		http.Error(w, "Contraseña incorrecta", http.StatusUnauthorized)
		return
	}

	user.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *LoginHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validar rol
	if user.Role != "customer" && user.Role != "delivery" {
		http.Error(w, "Rol inválido. Debe ser 'customer' o 'delivery'", http.StatusBadRequest)
		return
	}

	// Validar campos requeridos
	if user.Name == "" || user.Password == "" {
		http.Error(w, "Nombre y contraseña son requeridos", http.StatusBadRequest)
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