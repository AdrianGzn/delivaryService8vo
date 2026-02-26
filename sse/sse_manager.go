package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

func (m *SSEManager) RegisterClient(userId int) chan []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan []byte, 10)
	m.clients[userId] = ch
	log.Printf("Cliente %d registrado para SSE", userId)
	return ch
}

func (m *SSEManager) UnregisterClient(userId int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, exists := m.clients[userId]; exists {
		close(ch)
		delete(m.clients, userId)
		log.Printf("Cliente %d desconectado de SSE", userId)
	}
}

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

	select {
	case ch <- jsonData:
		log.Printf("Notificación enviada a usuario %d: %s", userId, event)
	default:
		log.Printf("Buffer lleno para usuario %d, mensaje descartado", userId)
	}

	return nil
}

func (m *SSEManager) NotifyOrderUpdate(order *models.Order) {
	m.NotifyUser(order.UserID, "order_update", order)

	if order.DeliveryID != nil {
		m.NotifyUser(*order.DeliveryID, "order_update", order)
	}

}

func (m *SSEManager) Broadcast(event string, data interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	message := struct {
		Event string      `json:"event"`
		Data  interface{} `json:"data"`
	}{
		Event: event,
		Data:  data,
	}

	jsonData, _ := json.Marshal(message)

	for userId, ch := range m.clients {
		select {
		case ch <- jsonData:
			log.Printf("Broadcast enviado a usuario %d", userId)
		default:
			log.Printf("Buffer lleno para usuario %d en broadcast", userId)
		}
	}
}

func (m *SSEManager) SSEHandler(w http.ResponseWriter, r *http.Request) {
	userIdStr := r.URL.Query().Get("userId")
	if userIdStr == "" {
		http.Error(w, "Se requiere userId en la query string", http.StatusBadRequest)
		return
	}

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		http.Error(w, "userId debe ser un número válido", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := m.RegisterClient(userId)
	defer m.UnregisterClient(userId)

	fmt.Fprintf(w, "event: connected\ndata: {\"userId\":%d,\"message\":\"Conectado al servicio de notificaciones\"}\n\n", userId)
	w.(http.Flusher).Flush()

	log.Printf("Usuario %d conectado a SSE", userId)

	// Mantener conexión abierta
	for {
		select {
		case <-r.Context().Done():
			log.Printf("Conexión SSE cerrada para usuario %d", userId)
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
