package handlers

import (
	"log"
	"net/http"
	"strconv"

	"deliveryService/websocket"

	"github.com/gorilla/mux"
	ws "github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	WebSocketManager *websocket.WebSocketManager
}

var upgrader = ws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIdStr := vars["userId"]

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		http.Error(w, "userId debe ser un numero valido", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar a WebSocket: %v", err)
		return
	}

	log.Printf("Usuario %d conectado a WebSocket", userId)
	h.WebSocketManager.RegisterClient(userId, conn)

	go func() {
		defer func() {
			h.WebSocketManager.UnregisterClient(userId, conn)
		}()

		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
					log.Printf("Error WebSocket usuario %d: %v", userId, err)
				}
				return
			}

			log.Printf("Mensaje recibido de usuario %d (tipo: %d): %s", userId, messageType, string(data))
		}
	}()

	welcome := map[string]interface{}{
		"event": "connected",
		"data": map[string]interface{}{
			"userId":  userId,
			"message": "Conectado al servicio de notificaciones",
		},
	}

	if err := conn.WriteJSON(welcome); err != nil {
		log.Printf("Error enviando bienvenida a usuario %d: %v", userId, err)
	}
}

func (h *WebSocketHandler) GetActiveConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	activeUsers := h.WebSocketManager.GetActiveUsers()
	w.Write([]byte(`{"activeUsers":` + strconv.Itoa(activeUsers) + `}`))
}
