package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"deliveryService/models"
)

type SSEManager struct {
	clients map[int]chan []byte
	mu      sync.RWMutex
}

func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[int]chan []byte),
	}
}

// Registrar un nuevo cliente SSE
func (m *SSEManager) RegisterClient(userId int) chan []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	ch := make(chan []byte, 10) // Buffer para evitar bloqueos
	m.clients[userId] = ch
	return ch
}

// Eliminar un cliente
func (m *SSEManager) UnregisterClient(userId int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if ch, exists := m.clients[userId]; exists {
		close(ch)
		delete(m.clients, userId)
	}
}

// Enviar notificación a un usuario específico
func (m *SSEManager) NotifyUser(userId int, event string, data interface{}) error {
	m.mu.RLock()
	ch, exists := m.clients[userId]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("usuario %d no está conectado", userId)
	}

	message := struct {
		Event string      `json:"event"`
		Data  interface{} `json:"data"`
	}{
		Event: event,
		Data:  data,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Enviar sin bloquear
	select {
	case ch <- jsonData:
	default:
		log.Printf("Buffer lleno para usuario %d, mensaje descartado", userId)
	}

	return nil
}

// Enviar notificación de actualización de orden
func (m *SSEManager) NotifyOrderUpdate(order *models.Order) {
	// Notificar al cliente que hizo el pedido
	m.NotifyUser(order.UserID, "order_update", order)
	
	// Si hay un repartidor asignado, también notificarlo
	if order.DeliveryID != nil {
		m.NotifyUser(*order.DeliveryID, "order_update", order)
	}
}


func (m *SSEManager) SSEHandler(w http.ResponseWriter, r *http.Request) {
	// Obtener userId del contexto (deberías setearlo en el middleware de auth)
	userId, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := m.RegisterClient(userId)
	defer m.UnregisterClient(userId)

	fmt.Fprintf(w, "event: connected\ndata: {\"message\":\"Conectado al servicio de notificaciones\"}\n\n")
	w.(http.Flusher).Flush()

	
	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			w.(http.Flusher).Flush()
		}
	}
}