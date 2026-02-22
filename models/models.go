package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Password string  `json:"-"`
	Role     string  `json:"role"` // "customer", "delivery"
	Address  *string `json:"address,omitempty"`
}

type Order struct {
	ID                int       `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Status            string    `json:"status"` // "pending", "pickup", "in_coming", "arrived", "delivered"
	EstablishmentName string    `json:"establishmentName"`
	EstablishmentAddr string    `json:"establishmentAddress"`
	Price             float64   `json:"price"`
	UserID            int       `json:"userId"`
	DeliveryID        *int      `json:"deliveryId,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func CreateTables(db *sql.DB) error {
	// Crear base de datos si no existe (opcional, normalmente ya está creada)
	_, err := db.Exec("CREATE DATABASE IF NOT EXISTS DeliveryService")
	if err != nil {
		return err
	}
	
	// Usar la base de datos
	_, err = db.Exec("USE DeliveryService")
	if err != nil {
		return err
	}

	// Tabla de usuarios - MySQL syntax
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		role ENUM('customer', 'delivery') NOT NULL,
		address TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	// Tabla de órdenes - MySQL syntax
	orderTable := `
	CREATE TABLE IF NOT EXISTS orders (
		id INT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT NOT NULL,
		status ENUM('pending', 'pickup', 'in_coming', 'arrived', 'delivered') NOT NULL,
		establishmentName VARCHAR(255) NOT NULL,
		establishmentAddress TEXT NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		user_id INT NOT NULL,
		delivery_id INT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (delivery_id) REFERENCES users(id) ON DELETE SET NULL,
		INDEX idx_user_id (user_id),
		INDEX idx_delivery_id (delivery_id),
		INDEX idx_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err = db.Exec(userTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(orderTable)
	return err
}

// Función para insertar datos de prueba
func SeedDatabase(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = db.Exec(`
			INSERT INTO users (name, password, role, address) VALUES 
			('cliente1', '123456', 'customer', 'Calle Cliente 123'),
			('cliente2', '123456', 'customer', 'Calle Cliente 456'),
			('repartidor1', '123456', 'delivery', NULL),
			('repartidor2', '123456', 'delivery', NULL)
		`)
		if err != nil {
			return err
		}
	}
	return nil
}