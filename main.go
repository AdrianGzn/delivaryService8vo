package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"deliveryService/handlers"
	"deliveryService/models"
	"deliveryService/websocket"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== INICIANDO DELIVERY SERVICE CON MYSQL ===")

	dsn := "adri:1234@tcp(100.30.88.139:3306)/DeliveryService?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error conectando a MySQL:", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()
	if err != nil {
		log.Fatal("Error haciendo ping a MySQL:", err)
	}
	log.Println("✅ Conectado a MySQL exitosamente")

	log.Println("Creando/verificando tablas...")
	err = models.CreateTables(db)
	if err != nil {
		log.Fatal("Error creando tablas:", err)
	}
	log.Println("✅ Tablas creadas/verificadas")

	err = models.SeedDatabase(db)
	if err != nil {
		log.Println("⚠️ Error insertando datos de prueba:", err)
	} else {
		log.Println("✅ Datos de prueba insertados")
	}

	log.Println("Inicializando WebSocket Manager...")
	wsManager := websocket.NewWebSocketManager()

	log.Println("Inicializando handlers...")
	userHandler := &handlers.UserHandler{DB: db}
	orderHandler := &handlers.OrderHandler{DB: db, WebSocketManager: wsManager}
	loginHandler := &handlers.LoginHandler{DB: db}
	wsHandler := &handlers.WebSocketHandler{WebSocketManager: wsManager}
	foodHandler := &handlers.FoodHandler{DB: db}

	log.Println("Configurando rutas...")
	r := mux.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("📡 %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/login", loginHandler.Login).Methods("POST", "OPTIONS")
	r.HandleFunc("/register", loginHandler.Register).Methods("POST", "OPTIONS")
	r.HandleFunc("/ws/{userId}", wsHandler.HandleWebSocket).Methods("GET", "OPTIONS")
	r.HandleFunc("/ws/status", wsHandler.GetActiveConnections).Methods("GET", "OPTIONS")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/users", userHandler.GetAllUsers).Methods("GET", "OPTIONS")
	api.HandleFunc("/users/{id}", userHandler.GetUser).Methods("GET", "OPTIONS")
	api.HandleFunc("/users/{id}", userHandler.UpdateUser).Methods("PUT", "OPTIONS")
	api.HandleFunc("/users/{id}", userHandler.DeleteUser).Methods("DELETE", "OPTIONS")

	api.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST", "OPTIONS")
	api.HandleFunc("/orders", orderHandler.GetAllOrders).Methods("GET", "OPTIONS")
	api.HandleFunc("/orders/user/{userId}", orderHandler.GetUserOrders).Methods("GET", "OPTIONS")
	api.HandleFunc("/orders/{id}", orderHandler.GetOrder).Methods("GET", "OPTIONS")
	api.HandleFunc("/orders/{id}/status", orderHandler.UpdateOrderStatus).Methods("PATCH", "OPTIONS")
	api.HandleFunc("/orders/{id}/assign", orderHandler.AssignDelivery).Methods("POST", "OPTIONS")
	api.HandleFunc("/orders/{id}", orderHandler.DeleteOrder).Methods("DELETE", "OPTIONS")

	api.HandleFunc("/food", foodHandler.CreateFood).Methods("POST", "OPTIONS")
	api.HandleFunc("/food", foodHandler.GetAllFood).Methods("GET", "OPTIONS")
	api.HandleFunc("/food/{id}", foodHandler.GetFood).Methods("GET", "OPTIONS")
	api.HandleFunc("/food/seller/{sellerId}", foodHandler.GetSellerFood).Methods("GET", "OPTIONS")
	api.HandleFunc("/food/{id}", foodHandler.UpdateFood).Methods("PUT", "OPTIONS")
	api.HandleFunc("/food/{id}", foodHandler.DeleteFood).Methods("DELETE", "OPTIONS")

	api.HandleFunc("/orders/{orderId}/items", orderHandler.GetOrderItems).Methods("GET", "OPTIONS")
	api.HandleFunc("/orders/{orderId}/items", orderHandler.AddOrderItem).Methods("POST", "OPTIONS")
	api.HandleFunc("/orders/{orderId}/items/{itemId}", orderHandler.RemoveOrderItem).Methods("DELETE", "OPTIONS")

	port := ":8080"
	log.Printf("🚀 Servidor corriendo en http://localhost%s", port)
	log.Println("📡 Endpoints disponibles:")
	log.Println("   - POST  /login")
	log.Println("   - POST  /register")
	log.Println("   - GET   /ws/{userId} (WebSocket)")
	log.Println("   - GET   /ws/status")
	log.Println("   - GET   /health")
	log.Println("   - GET   /api/users")
	log.Println("   - PUT   /api/users/{id}")
	log.Println("   - DELETE /api/users/{id}")
	log.Println("   - GET   /api/orders")
	log.Println("   - POST  /api/orders")
	log.Println("   - GET   /api/orders/{id}")
	log.Println("   - GET   /api/orders/user/{userId}")
	log.Println("   - PATCH /api/orders/{id}/status")
	log.Println("   - POST  /api/orders/{id}/assign")
	log.Println("   - DELETE /api/orders/{id}")
	log.Println("   - GET   /api/food")
	log.Println("   - POST  /api/food")
	log.Println("   - GET   /api/food/{id}")
	log.Println("   - GET   /api/food/seller/{sellerId}")
	log.Println("   - PUT   /api/food/{id}")
	log.Println("   - DELETE /api/food/{id}")
	log.Println("   - GET   /api/orders/{orderId}/items")
	log.Println("   - POST  /api/orders/{orderId}/items")
	log.Println("   - DELETE /api/orders/{orderId}/items/{itemId}")
	log.Println("Presiona Ctrl+C para detener el servidor")

	log.Fatal(http.ListenAndServe(port, r))
}