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

	// Obtener userId del contexto (usuario autenticado)
	userId, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Usuario no autenticado", http.StatusUnauthorized)
		return
	}

	// Validar que el rol sea customer (esto debería hacerse en middleware)
	role, _ := r.Context().Value("user_role").(string)
	if role != "customer" {
		http.Error(w, "Solo los clientes pueden crear órdenes", http.StatusForbidden)
		return
	}

	order.UserID = userId
	order.Status = "pending"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	result, err := h.DB.Exec(
		`INSERT INTO orders (title, description, status, establishmentName, 
			establishmentAddress, price, user_id, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		order.Title, order.Description, order.Status, order.EstablishmentName,
		order.EstablishmentAddr, order.Price, order.UserID, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "Error al crear orden: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	order.ID = int(id)

	// Notificar al cliente que su orden fue creada
	h.SSEManager.NotifyOrderUpdate(&order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
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

	// Verificar que el usuario tenga permiso para ver esta orden
	userId := r.Context().Value("user_id").(int)
	role := r.Context().Value("user_role").(string)

	if role != "delivery" && order.UserID != userId {
		http.Error(w, "No tiene permiso para ver esta orden", http.StatusForbidden)
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
	}
	err = json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validar status válido
	validStatus := map[string]bool{
		"pending": true, "pickup": true, "in_coming": true, 
		"arrived": true, "delivered": true,
	}
	if !validStatus[updateData.Status] {
		http.Error(w, "Status inválido", http.StatusBadRequest)
		return
	}

	// Obtener la orden actual para verificar permisos
	var currentOrder models.Order
	err = h.DB.QueryRow("SELECT user_id, delivery_id FROM orders WHERE id = ?", id).
		Scan(&currentOrder.UserID, &currentOrder.DeliveryID)
	if err != nil {
		http.Error(w, "Orden no encontrada", http.StatusNotFound)
		return
	}

	// Verificar permisos (solo el repartidor asignado puede actualizar)
	userId := r.Context().Value("user_id").(int)
	role := r.Context().Value("user_role").(string)

	if role != "delivery" || (currentOrder.DeliveryID != nil && *currentOrder.DeliveryID != userId) {
		http.Error(w, "No tiene permiso para actualizar esta orden", http.StatusForbidden)
		return
	}

	// Actualizar status
	_, err = h.DB.Exec(
		"UPDATE orders SET status = ?, updated_at = ? WHERE id = ?",
		updateData.Status, time.Now(), id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener la orden actualizada para notificar
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
		// Notificar al cliente sobre el cambio de estado
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

	// Verificar que el delivery exista y sea repartidor
	var role string
	err = h.DB.QueryRow("SELECT role FROM users WHERE id = ?", assignData.DeliveryID).Scan(&role)
	if err != nil || role != "delivery" {
		http.Error(w, "Repartidor no válido", http.StatusBadRequest)
		return
	}

	// Asignar repartidor y cambiar status a "pickup"
	_, err = h.DB.Exec(
		"UPDATE orders SET delivery_id = ?, status = 'pickup', updated_at = ? WHERE id = ?",
		assignData.DeliveryID, time.Now(), id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener orden actualizada
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
		// Notificar a ambos sobre la asignación
		h.SSEManager.NotifyOrderUpdate(&updatedOrder)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedOrder)
}

func (h *OrderHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("user_id").(int)
	role := r.Context().Value("user_role").(string)

	var rows *sql.Rows
	var err error

	if role == "customer" {
		// Cliente ve sus órdenes
		rows, err = h.DB.Query(`
			SELECT id, title, description, status, establishmentName, 
				   establishmentAddress, price, user_id, delivery_id, created_at, updated_at 
			FROM orders WHERE user_id = ? ORDER BY created_at DESC`, userId)
	} else {
		// Repartidor ve órdenes asignadas o disponibles
		rows, err = h.DB.Query(`
			SELECT id, title, description, status, establishmentName, 
				   establishmentAddress, price, user_id, delivery_id, created_at, updated_at 
			FROM orders WHERE delivery_id = ? OR (delivery_id IS NULL AND status = 'pending')
			ORDER BY created_at DESC`, userId)
	}

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

func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	// Solo el admin o el cliente que creó la orden puede eliminarla (pendiente)
	_, err = h.DB.Exec("DELETE FROM orders WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}