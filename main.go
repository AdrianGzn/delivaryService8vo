package main

import (
	"database/sql"
	"log"
	"net/http"

	"deliveryService/handlers"
	"deliveryService/models"
	"deliveryService/sse"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Inicializar base de datos SQLite
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

	
	userHandler := &handlers.UserHandler{DB: db}
	orderHandler := &handlers.OrderHandler{DB: db, SSEManager: sseManager}
	loginHandler := &handlers.LoginHandler{DB: db}

	
	r := mux.NewRouter()

	// login/registro
	r.HandleFunc("/login", loginHandler.Login).Methods("POST")
	r.HandleFunc("/register", loginHandler.Register).Methods("POST")
	
	// Ruta SSE
	r.HandleFunc("/sse", sseManager.SSEHandler).Methods("GET")

	api := r.PathPrefix("/api").Subrouter()
	
	// User routes
	api.HandleFunc("/users", userHandler.CreateUser).Methods("POST")
	api.HandleFunc("/users", userHandler.GetAllUsers).Methods("GET")
	api.HandleFunc("/users/{id}", userHandler.GetUser).Methods("GET")
	api.HandleFunc("/users/{id}", userHandler.UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", userHandler.DeleteUser).Methods("DELETE")

	// Order routes
	api.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")
	api.HandleFunc("/orders", orderHandler.GetAllOrders).Methods("GET")
	api.HandleFunc("/orders/user/{userId}", orderHandler.GetUserOrders).Methods("GET")
	api.HandleFunc("/orders/{id}", orderHandler.GetOrder).Methods("GET")
	api.HandleFunc("/orders/{id}/status", orderHandler.UpdateOrderStatus).Methods("PATCH")
	api.HandleFunc("/orders/{id}/assign", orderHandler.AssignDelivery).Methods("POST")
	api.HandleFunc("/orders/{id}", orderHandler.DeleteOrder).Methods("DELETE")

	log.Println("Servidor iniciado en :8080 (modo público - sin autenticación)")
	log.Fatal(http.ListenAndServe(":8080", r))
}