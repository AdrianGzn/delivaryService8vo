package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"deliveryService/models"
	"deliveryService/websocket"

	"github.com/gorilla/mux"
)

type OrderHandler struct {
	DB               *sql.DB
	WebSocketManager *websocket.WebSocketManager
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.CreateOrderRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Error al decodificar JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "Debe incluir al menos 1 item de comida", http.StatusBadRequest)
		return
	}

	if req.SellerID == 0 {
		http.Error(w, "sellerId es requerido", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 {
		req.UserID = 1
	}

	var totalPrice float64
	for _, item := range req.Items {
		var foodPrice float64
		var sellerId int
		err := h.DB.QueryRow(
			"SELECT price, seller_id FROM food WHERE id = ?",
			item.FoodID,
		).Scan(&foodPrice, &sellerId)

		if err == sql.ErrNoRows {
			http.Error(w, "Alimento con ID "+strconv.Itoa(item.FoodID)+" no encontrado", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "Error al consultar alimento: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if sellerId != req.SellerID {
			http.Error(w, "El alimento ID "+strconv.Itoa(item.FoodID)+" no pertenece a este vendedor", http.StatusBadRequest)
			return
		}

		totalPrice += foodPrice * float64(item.Quantity)
	}

	order := models.Order{
		Title:       "Orden de comida",
		Description: "Orden de múltiples items",
		Status:      "pending",
		Price:       totalPrice,
		UserID:      req.UserID,
		SellerID:    req.SellerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result, err := h.DB.Exec(
		`INSERT INTO orders (title, description, status, price, user_id, seller_id, delivery_id, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		order.Title, order.Description, order.Status, order.Price, order.UserID, order.SellerID, nil,
		order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "Error al crear orden: "+err.Error(), http.StatusInternalServerError)
		return
	}

	orderId, _ := result.LastInsertId()
	order.ID = int(orderId)

	// 🔹 OBTENER LA ORDEN COMPLETA DESPUÉS DE CREARLA
	err = h.DB.QueryRow(`
		SELECT id, title, description, status, price, user_id, seller_id, delivery_id, created_at, updated_at 
		FROM orders WHERE id = ?`, orderId,
	).Scan(&order.ID, &order.Title, &order.Description, &order.Status,
		&order.Price, &order.UserID, &order.SellerID, &order.DeliveryID, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		http.Error(w, "Error al obtener orden creada: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var items []models.OrderItem
	for _, item := range req.Items {
		var foodName string
		var foodPrice float64
		err := h.DB.QueryRow(
			"SELECT name, price FROM food WHERE id = ?",
			item.FoodID,
		).Scan(&foodName, &foodPrice)

		if err != nil {
			continue
		}

		itemResult, err := h.DB.Exec(
			`INSERT INTO order_items (order_id, food_id, quantity, price) VALUES (?, ?, ?, ?)`,
			order.ID, item.FoodID, item.Quantity, foodPrice,
		)
		if err == nil {
			itemId, _ := itemResult.LastInsertId()
			items = append(items, models.OrderItem{
				ID:       int(itemId),
				OrderID:  order.ID,
				FoodID:   item.FoodID,
				FoodName: foodName,
				Quantity: item.Quantity,
				Price:    foodPrice,
			})
		}
	}

	h.WebSocketManager.NotifyOrderUpdate(&order)

	orderDetail := models.OrderDetail{
		Order: order,
		Items: items,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(orderDetail)
}

func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, title, description, status, price, user_id, seller_id, delivery_id, created_at, updated_at 
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
			&order.Price, &order.UserID, &order.SellerID, &order.DeliveryID, &order.CreatedAt, &order.UpdatedAt)
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
		SELECT id, title, description, status, price, user_id, seller_id, delivery_id, created_at, updated_at 
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
			&order.Price, &order.UserID, &order.SellerID, &order.DeliveryID, &order.CreatedAt, &order.UpdatedAt)
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
		SELECT id, title, description, status, price, user_id, seller_id, delivery_id, created_at, updated_at 
		FROM orders WHERE id = ?`, id,
	).Scan(&order.ID, &order.Title, &order.Description, &order.Status,
		&order.Price, &order.UserID, &order.SellerID, &order.DeliveryID, &order.CreatedAt, &order.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Orden no encontrada", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := h.DB.Query(`
		SELECT oi.id, oi.order_id, oi.food_id, f.name, oi.quantity, oi.price 
		FROM order_items oi
		JOIN food f ON oi.food_id = f.id
		WHERE oi.order_id = ?
		ORDER BY oi.created_at DESC`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.FoodID, &item.FoodName, &item.Quantity, &item.Price)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	orderDetail := models.OrderDetail{
		Order: order,
		Items: items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orderDetail)
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

	validStatus := map[string]bool{
		"pending": true, "pickup": true, "in_coming": true,
		"arrived": true, "delivered": true,
	}
	if !validStatus[updateData.Status] {
		http.Error(w, "Estado inválido", http.StatusBadRequest)
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
		SELECT id, title, description, status, price, user_id, seller_id, delivery_id, created_at, updated_at 
		FROM orders WHERE id = ?`, id,
	).Scan(&updatedOrder.ID, &updatedOrder.Title, &updatedOrder.Description,
		&updatedOrder.Status, &updatedOrder.Price, &updatedOrder.UserID, &updatedOrder.SellerID,
		&updatedOrder.DeliveryID, &updatedOrder.CreatedAt, &updatedOrder.UpdatedAt)

	if err == nil {
		h.WebSocketManager.NotifyOrderUpdate(&updatedOrder)
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

	var role string
	err = h.DB.QueryRow("SELECT role FROM users WHERE id = ?", assignData.DeliveryID).Scan(&role)
	if err != nil || role != "delivery" {
		http.Error(w, "Repartidor inválido", http.StatusBadRequest)
		return
	}

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
		SELECT id, title, description, status, price, user_id, seller_id, delivery_id, created_at, updated_at 
		FROM orders WHERE id = ?`, id,
	).Scan(&updatedOrder.ID, &updatedOrder.Title, &updatedOrder.Description,
		&updatedOrder.Status, &updatedOrder.Price, &updatedOrder.UserID, &updatedOrder.SellerID,
		&updatedOrder.DeliveryID, &updatedOrder.CreatedAt, &updatedOrder.UpdatedAt)

	if err == nil {
		h.WebSocketManager.NotifyOrderUpdate(&updatedOrder)
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

	_, err = h.DB.Exec("DELETE FROM orders WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) GetOrderItems(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderId, err := strconv.Atoi(vars["orderId"])
	if err != nil {
		http.Error(w, "orderId inválido", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(`
		SELECT oi.id, oi.order_id, oi.food_id, f.name, oi.quantity, oi.price 
		FROM order_items oi
		JOIN food f ON oi.food_id = f.id
		WHERE oi.order_id = ?
		ORDER BY oi.created_at DESC`, orderId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.FoodID, &item.FoodName, &item.Quantity, &item.Price)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *OrderHandler) AddOrderItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderId, err := strconv.Atoi(vars["orderId"])
	if err != nil {
		http.Error(w, "orderId inválido", http.StatusBadRequest)
		return
	}

	var req struct {
		FoodID   int `json:"foodId"`
		Quantity int `json:"quantity"`
	}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
		return
	}

	if req.FoodID == 0 || req.Quantity == 0 {
		http.Error(w, "foodId y quantity son requeridos", http.StatusBadRequest)
		return
	}

	var orderSellerId int
	err = h.DB.QueryRow("SELECT seller_id FROM orders WHERE id = ?", orderId).Scan(&orderSellerId)
	if err == sql.ErrNoRows {
		http.Error(w, "Orden no encontrada", http.StatusNotFound)
		return
	}

	var foodName string
	var foodPrice float64
	var foodSellerId int
	err = h.DB.QueryRow(
		"SELECT name, price, seller_id FROM food WHERE id = ?",
		req.FoodID,
	).Scan(&foodName, &foodPrice, &foodSellerId)

	if err == sql.ErrNoRows {
		http.Error(w, "Alimento no encontrado", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Error al consultar alimento: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if foodSellerId != orderSellerId {
		http.Error(w, "El alimento no pertenece a este vendedor", http.StatusBadRequest)
		return
	}

	result, err := h.DB.Exec(
		`INSERT INTO order_items (order_id, food_id, quantity, price) VALUES (?, ?, ?, ?)`,
		orderId, req.FoodID, req.Quantity, foodPrice,
	)
	if err != nil {
		http.Error(w, "Error al agregar item: "+err.Error(), http.StatusInternalServerError)
		return
	}

	itemId, _ := result.LastInsertId()
	item := models.OrderItem{
		ID:       int(itemId),
		OrderID:  orderId,
		FoodID:   req.FoodID,
		FoodName: foodName,
		Quantity: req.Quantity,
		Price:    foodPrice,
	}

	var newTotal float64
	err = h.DB.QueryRow(
		"SELECT SUM(quantity * price) FROM order_items WHERE order_id = ?",
		orderId,
	).Scan(&newTotal)

	if err == nil {
		h.DB.Exec("UPDATE orders SET price = ?, updated_at = ? WHERE id = ?", newTotal, time.Now(), orderId)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *OrderHandler) RemoveOrderItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderId, err := strconv.Atoi(vars["orderId"])
	itemId, err2 := strconv.Atoi(vars["itemId"])

	if err != nil || err2 != nil {
		http.Error(w, "orderId o itemId inválido", http.StatusBadRequest)
		return
	}

	var exists int
	err = h.DB.QueryRow("SELECT 1 FROM order_items WHERE id = ? AND order_id = ?", itemId, orderId).Scan(&exists)
	if err == sql.ErrNoRows {
		http.Error(w, "Item no encontrado en esta orden", http.StatusNotFound)
		return
	}

	_, err = h.DB.Exec("DELETE FROM order_items WHERE id = ? AND order_id = ?", itemId, orderId)
	if err != nil {
		http.Error(w, "Error al remover item: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var newTotal sql.NullFloat64
	err = h.DB.QueryRow(
		"SELECT SUM(quantity * price) FROM order_items WHERE order_id = ?",
		orderId,
	).Scan(&newTotal)

	if err == nil {
		if newTotal.Valid {
			h.DB.Exec("UPDATE orders SET price = ?, updated_at = ? WHERE id = ?", newTotal.Float64, time.Now(), orderId)
		} else {
			h.DB.Exec("UPDATE orders SET price = 0, updated_at = ? WHERE id = ?", time.Now(), orderId)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}