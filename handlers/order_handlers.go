package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"deliveryService/models"
	"deliveryService/sse"
	"github.com/gorilla/mux"
)

type OrderHandler struct {
	DB        *sql.DB
	SSEManager *sse.SSEManager
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	
	if order.Title == "" || order.Description == "" || order.EstablishmentName == "" {
		http.Error(w, "Faltan campos requeridos", http.StatusBadRequest)
		return
	}

	
	if order.UserID == 0 {
		order.UserID = 1
	}

	order.Status = "pending"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	result, err := h.DB.Exec(
		`INSERT INTO orders (title, description, status, establishmentName, 
			establishmentAddress, price, user_id, delivery_id, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		order.Title, order.Description, order.Status, order.EstablishmentName,
		order.EstablishmentAddr, order.Price, order.UserID, order.DeliveryID, 
		order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "Error al crear orden: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	order.ID = int(id)

	// Notificar al cliente
	h.SSEManager.NotifyOrderUpdate(&order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, title, description, status, establishmentName, 
			   establishmentAddress, price, user_id, delivery_id, created_at, updated_at 
		FROM orders ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.Title, &order.Description, &order.Status,
			&order.EstablishmentName, &order.EstablishmentAddr, &order.Price,
			&order.UserID, &order.DeliveryID, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *OrderHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, err := strconv.Atoi(vars["userId"])
	if err != nil {
		http.Error(w, "userId inválido", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, title, description, status, establishmentName, 
			   establishmentAddress, price, user_id, delivery_id, created_at, updated_at 
		FROM orders WHERE user_id = ? OR delivery_id = ?
		ORDER BY created_at DESC`, userId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.Title, &order.Description, &order.Status,
			&order.EstablishmentName, &order.EstablishmentAddr, &order.Price,
			&order.UserID, &order.DeliveryID, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var order models.Order
	err = h.DB.QueryRow(`
		SELECT id, title, description, status, establishmentName, 
			   establishmentAddress, price, user_id, delivery_id, created_at, updated_at 
		FROM orders WHERE id = ?`, id,
	).Scan(&order.ID, &order.Title, &order.Description, &order.Status, 
		&order.EstablishmentName, &order.EstablishmentAddr, &order.Price, 
		&order.UserID, &order.DeliveryID, &order.CreatedAt, &order.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Orden no encontrada", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var updateData struct {
		Status string `json:"status"`
		UserID int    `json:"userId"`
	}
	err = json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	
	validStatus := map[string]bool{
		"pending": true, "pickup": true, "in_coming": true, 
		"arrived": true, "delivered": true,
	}
	if !validStatus[updateData.Status] {
		http.Error(w, "Status inválido", http.StatusBadRequest)
		return
	}

	
	_, err = h.DB.Exec(
		"UPDATE orders SET status = ?, updated_at = ? WHERE id = ?",
		updateData.Status, time.Now(), id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	
	var updatedOrder models.Order
	err = h.DB.QueryRow(`
		SELECT id, title, description, status, establishmentName, 
			   establishmentAddress, price, user_id, delivery_id, created_at, updated_at 
		FROM orders WHERE id = ?`, id,
	).Scan(&updatedOrder.ID, &updatedOrder.Title, &updatedOrder.Description, 
		&updatedOrder.Status, &updatedOrder.EstablishmentName, 
		&updatedOrder.EstablishmentAddr, &updatedOrder.Price, 
		&updatedOrder.UserID, &updatedOrder.DeliveryID, 
		&updatedOrder.CreatedAt, &updatedOrder.UpdatedAt)

	if err == nil {
		h.SSEManager.NotifyOrderUpdate(&updatedOrder)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedOrder)
}

func (h *OrderHandler) AssignDelivery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var assignData struct {
		DeliveryID int `json:"deliveryId"`
	}
	err = json.NewDecoder(r.Body).Decode(&assignData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verificar que el delivery exista
	var role string
	err = h.DB.QueryRow("SELECT role FROM users WHERE id = ?", assignData.DeliveryID).Scan(&role)
	if err != nil || role != "delivery" {
		http.Error(w, "Repartidor no válido", http.StatusBadRequest)
		return
	}

	// Asignar repartidor
	_, err = h.DB.Exec(
		"UPDATE orders SET delivery_id = ?, status = 'pickup', updated_at = ? WHERE id = ?",
		assignData.DeliveryID, time.Now(), id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	
	var updatedOrder models.Order
	err = h.DB.QueryRow(`
		SELECT id, title, description, status, establishmentName, 
			   establishmentAddress, price, user_id, delivery_id, created_at, updated_at 
		FROM orders WHERE id = ?`, id,
	).Scan(&updatedOrder.ID, &updatedOrder.Title, &updatedOrder.Description, 
		&updatedOrder.Status, &updatedOrder.EstablishmentName, 
		&updatedOrder.EstablishmentAddr, &updatedOrder.Price, 
		&updatedOrder.UserID, &updatedOrder.DeliveryID, 
		&updatedOrder.CreatedAt, &updatedOrder.UpdatedAt)

	if err == nil {
		h.SSEManager.NotifyOrderUpdate(&updatedOrder)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedOrder)
}

func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	
	var order models.Order
	h.DB.QueryRow("SELECT user_id, delivery_id FROM orders WHERE id = ?", id).Scan(&order.UserID, &order.DeliveryID)

	_, err = h.DB.Exec("DELETE FROM orders WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	
	if order.UserID != 0 {
		h.SSEManager.NotifyUser(order.UserID, "order_deleted", map[string]int{"id": id})
	}
	if order.DeliveryID != nil {
		h.SSEManager.NotifyUser(*order.DeliveryID, "order_deleted", map[string]int{"id": id})
	}

	w.WriteHeader(http.StatusNoContent)
}