package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"deliveryService/models"

	"github.com/gorilla/mux"
)

type FoodHandler struct {
	DB *sql.DB
}

func (h *FoodHandler) CreateFood(w http.ResponseWriter, r *http.Request) {
	var food models.Food
	err := json.NewDecoder(r.Body).Decode(&food)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if food.Name == "" || food.SellerID == 0 || food.Price == 0 {
		http.Error(w, "Faltan campos requeridos", http.StatusBadRequest)
		return
	}

	// Verificar que el seller exista
	var role string
	err = h.DB.QueryRow("SELECT role FROM users WHERE id = ?", food.SellerID).Scan(&role)
	if err != nil || role != "seller" {
		http.Error(w, "Seller inválido", http.StatusBadRequest)
		return
	}

	result, err := h.DB.Exec(
		"INSERT INTO food (seller_id, name, price) VALUES (?, ?, ?)",
		food.SellerID, food.Name, food.Price,
	)
	if err != nil {
		http.Error(w, "Error al crear alimento: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	food.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(food)
}

func (h *FoodHandler) GetFood(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var food models.Food
	err = h.DB.QueryRow(
		"SELECT id, seller_id, name, price FROM food WHERE id = ?",
		id,
	).Scan(&food.ID, &food.SellerID, &food.Name, &food.Price)

	if err == sql.ErrNoRows {
		http.Error(w, "Alimento no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(food)
}

func (h *FoodHandler) GetAllFood(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, seller_id, name, price FROM food ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var foods []models.Food
	for rows.Next() {
		var food models.Food
		err := rows.Scan(&food.ID, &food.SellerID, &food.Name, &food.Price)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		foods = append(foods, food)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foods)
}

func (h *FoodHandler) GetSellerFood(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sellerId, err := strconv.Atoi(vars["sellerId"])
	if err != nil {
		http.Error(w, "sellerId inválido", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(
		"SELECT id, seller_id, name, price FROM food WHERE seller_id = ? ORDER BY id DESC",
		sellerId,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var foods []models.Food
	for rows.Next() {
		var food models.Food
		err := rows.Scan(&food.ID, &food.SellerID, &food.Name, &food.Price)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		foods = append(foods, food)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foods)
}

func (h *FoodHandler) UpdateFood(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var food models.Food
	err = json.NewDecoder(r.Body).Decode(&food)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if food.Name == "" || food.Price == 0 {
		http.Error(w, "Faltan campos requeridos", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(
		"UPDATE food SET name = ?, price = ? WHERE id = ?",
		food.Name, food.Price, id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener el alimento actualizado
	err = h.DB.QueryRow(
		"SELECT id, seller_id, name, price FROM food WHERE id = ?",
		id,
	).Scan(&food.ID, &food.SellerID, &food.Name, &food.Price)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(food)
}

func (h *FoodHandler) DeleteFood(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec("DELETE FROM food WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
