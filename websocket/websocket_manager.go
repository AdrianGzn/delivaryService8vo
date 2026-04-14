package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"

	"deliveryService/models"
)

type WebSocketManager struct {
	clients   map[int]map[*websocket.Conn]bool // userId -> {conexiones}
	mu        sync.RWMutex
	broadcast chan []byte
}

type WSMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

func NewWebSocketManager() *WebSocketManager {
	manager := &WebSocketManager{
		clients:   make(map[int]map[*websocket.Conn]bool),
		broadcast: make(chan []byte, 100),
	}
	go manager.handleBroadcast()
	return manager
}

// RegisterClient registra una nueva conexión WebSocket para un usuario
func (m *WebSocketManager) RegisterClient(userId int, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clients[userId] == nil {
		m.clients[userId] = make(map[*websocket.Conn]bool)
	}
	m.clients[userId][conn] = true
	log.Printf("Cliente WebSocket registrado: userID=%d, total conexiones para este user=%d", userId, len(m.clients[userId]))
}

// UnregisterClient desregistra una conexión WebSocket
func (m *WebSocketManager) UnregisterClient(userId int, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clients[userId] != nil {
		delete(m.clients[userId], conn)
		if len(m.clients[userId]) == 0 {
			delete(m.clients, userId)
			log.Printf("Usuario %d completamente desconectado", userId)
		} else {
			log.Printf("Cliente desconectado: userID=%d, conexiones restantes=%d", userId, len(m.clients[userId]))
		}
	}
	conn.Close()
}

// NotifyUser envía un mensaje a un usuario específico
func (m *WebSocketManager) NotifyUser(userId int, event string, data interface{}) error {
	m.mu.RLock()
	conns, exists := m.clients[userId]
	m.mu.RUnlock()

	if !exists || len(conns) == 0 {
		return fmt.Errorf("usuario %d no conectado", userId)
	}

	message := WSMessage{
		Event: event,
		Data:  data,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Enviar a todas las conexiones del usuario
	m.mu.RLock()
	defer m.mu.RUnlock()

	for conn := range conns {
		err := conn.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			log.Printf("Error enviando mensaje a usuario %d: %v", userId, err)
			// No cerramos aquí, se cerrará cuando se detecte la desconexión
			continue
		}
	}

	log.Printf("Notificación enviada a usuario %d: evento=%s", userId, event)
	return nil
}

// NotifyOrderUpdate notifica a los usuarios relacionados con una orden
func (m *WebSocketManager) NotifyOrderUpdate(order *models.Order) {
	// Notificamos globalmente para que los repartidores puedan ver los nuevos pedidos
	m.Broadcast("order_update", order)
}

// Broadcast envía un mensaje a todos los usuarios conectados
func (m *WebSocketManager) Broadcast(event string, data interface{}) {
	message := WSMessage{
		Event: event,
		Data:  data,
	}

	jsonData, _ := json.Marshal(message)
	m.broadcast <- jsonData
}

// handleBroadcast maneja los mensajes de broadcast
func (m *WebSocketManager) handleBroadcast() {
	for msg := range m.broadcast {
		m.mu.RLock()
		for userId, conns := range m.clients {
			for conn := range conns {
				err := conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					log.Printf("Error en broadcast a usuario %d: %v", userId, err)
				}
			}
		}
		m.mu.RUnlock()
	}
}

// GetActiveUsers retorna el número de usuarios conectados
func (m *WebSocketManager) GetActiveUsers() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}
