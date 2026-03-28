package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"deliveryService/models"

	"github.com/gorilla/mux"
)

type UserHandler struct {
	DB *sql.DB
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validar rol
	validRoles := map[string]bool{"customer": true, "delivery": true, "seller": true}
	if !validRoles[user.Role] {
		http.Error(w, "Rol inválido. Debe ser 'customer', 'delivery' o 'seller'", http.StatusBadRequest)
		return
	}

	// Validar que sellers requieren establishmentName y establishmentAddress
	if user.Role == "seller" {
		if user.EstablishmentName == nil || *user.EstablishmentName == "" ||
			user.EstablishmentAddr == nil || *user.EstablishmentAddr == "" {
			http.Error(w, "Los sellers deben proporcionar establishmentName y establishmentAddress", http.StatusBadRequest)
			return
		}
	}

	// En producción, hashear el password
	result, err := h.DB.Exec(
		"INSERT INTO users (name, password, role, address, establishmentName, establishmentAddress) VALUES (?, ?, ?, ?, ?, ?)",
		user.Name, user.Password, user.Role, user.Address, user.EstablishmentName, user.EstablishmentAddr,
	)
	if err != nil {
		http.Error(w, "Error al crear usuario: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	user.ID = int(id)
	user.Password = "" // No enviar password en respuesta

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var user models.User
	err = h.DB.QueryRow(
		"SELECT id, name, role, address, establishmentName, establishmentAddress FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Name, &user.Role, &user.Address, &user.EstablishmentName, &user.EstablishmentAddr)

	if err == sql.ErrNoRows {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var user models.User
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(
		"UPDATE users SET name = ?, address = ?, establishmentName = ?, establishmentAddress = ? WHERE id = ?",
		user.Name, user.Address, user.EstablishmentName, user.EstablishmentAddr, id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.ID = id
	user.Password = ""
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, name, role, address, establishmentName, establishmentAddress FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Name, &user.Role, &user.Address, &user.EstablishmentName, &user.EstablishmentAddr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
