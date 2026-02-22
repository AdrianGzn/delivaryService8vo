package main

import (
	"database/sql"
	"log"
	"net/http"

	"deliveryService/handlers"
	"deliveryService/middleware"
	"deliveryService/models"
	"deliveryService/sse"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./delivery.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	
	err = models.CreateTables(db)
	if err != nil {
		log.Fatal(err)
	}

	
	sseManager := sse.NewSSEManager()
	authMiddleware := &middleware.AuthMiddleware{}

	
	userHandler := &handlers.UserHandler{DB: db}
	orderHandler := &handlers.OrderHandler{DB: db, SSEManager: sseManager}
	loginHandler := &handlers.LoginHandler{DB: db, JWTSecret: "tu-secreto-super-seguro"}

	r := mux.NewRouter()
	// Rutas públicas
	r.HandleFunc("/api/login", loginHandler.Login).Methods("POST")
	r.HandleFunc("/api/register", loginHandler.Register).Methods("POST")
	
	// Ruta SSE (requiere autenticación)
	r.HandleFunc("/api/sse", authMiddleware.Authenticate(sseManager.SSEHandler, "customer", "delivery")).Methods("GET")

	// API Routes
	api := r.PathPrefix("/api").Subrouter()

	// User routes (protegidas)
	api.HandleFunc("/users", authMiddleware.Authenticate(userHandler.CreateUser, "customer", "delivery")).Methods("POST")
	api.HandleFunc("/users", authMiddleware.Authenticate(userHandler.GetAllUsers, "customer", "delivery")).Methods("GET")
	api.HandleFunc("/users/{id}", authMiddleware.Authenticate(userHandler.GetUser, "customer", "delivery")).Methods("GET")
	api.HandleFunc("/users/{id}", authMiddleware.Authenticate(userHandler.UpdateUser, "customer", "delivery")).Methods("PUT")
	api.HandleFunc("/users/{id}", authMiddleware.Authenticate(userHandler.DeleteUser, "customer", "delivery")).Methods("DELETE")

	// Order routes (protegidas)
	api.HandleFunc("/orders", authMiddleware.Authenticate(orderHandler.CreateOrder, "customer")).Methods("POST")
	api.HandleFunc("/orders", authMiddleware.Authenticate(orderHandler.GetUserOrders, "customer", "delivery")).Methods("GET")
	api.HandleFunc("/orders/{id}", authMiddleware.Authenticate(orderHandler.GetOrder, "customer", "delivery")).Methods("GET")
	api.HandleFunc("/orders/{id}/status", authMiddleware.Authenticate(orderHandler.UpdateOrderStatus, "delivery")).Methods("PATCH")
	api.HandleFunc("/orders/{id}/assign", authMiddleware.Authenticate(orderHandler.AssignDelivery, "delivery")).Methods("POST")
	api.HandleFunc("/orders/{id}", authMiddleware.Authenticate(orderHandler.DeleteOrder, "customer")).Methods("DELETE")

	
	log.Println("Servidor iniciado en :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}