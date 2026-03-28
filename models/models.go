package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Password          string  `json:"-"`
	Role              string  `json:"role"` // "customer", "delivery", "seller"
	Address           *string `json:"address,omitempty"`
	EstablishmentName *string `json:"establishmentName,omitempty"`
	EstablishmentAddr *string `json:"establishmentAddress,omitempty"`
}

type Order struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // "pending", "pickup", "in_coming", "arrived", "delivered"
	Price       float64   `json:"price"`
	UserID      int       `json:"userId"`
	SellerID    int       `json:"sellerId"`
	DeliveryID  *int      `json:"deliveryId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Food struct {
	ID       int     `json:"id"`
	SellerID int     `json:"sellerId"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
}

type OrderItem struct {
	ID       int     `json:"id"`
	OrderID  int     `json:"orderId"`
	FoodID   int     `json:"foodId"`
	FoodName string  `json:"foodName"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"` // Precio al momento de la compra
}

type OrderDetail struct {
	Order Order       `json:"order"`
	Items []OrderItem `json:"items"`
}

type CreateOrderRequest struct {
	UserID   int `json:"userId"`
	SellerID int `json:"sellerId"`
	Items    []struct {
		FoodID   int `json:"foodId"`
		Quantity int `json:"quantity"`
	} `json:"items"`
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
		role ENUM('customer', 'delivery', 'seller') NOT NULL,
		address TEXT,
		establishmentName VARCHAR(255),
		establishmentAddress TEXT,
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
		price DECIMAL(10,2) NOT NULL,
		user_id INT NOT NULL,
		seller_id INT NOT NULL,
		delivery_id INT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (delivery_id) REFERENCES users(id) ON DELETE SET NULL,
		INDEX idx_user_id (user_id),
		INDEX idx_seller_id (seller_id),
		INDEX idx_delivery_id (delivery_id),
		INDEX idx_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	// Tabla de comidas - MySQL syntax
	foodTable := `
	CREATE TABLE IF NOT EXISTS food (
		id INT AUTO_INCREMENT PRIMARY KEY,
		seller_id INT NOT NULL,
		name VARCHAR(255) NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE CASCADE,
		INDEX idx_seller_id (seller_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	// Tabla de items de órdenes
	orderItemsTable := `
	CREATE TABLE IF NOT EXISTS order_items (
		id INT AUTO_INCREMENT PRIMARY KEY,
		order_id INT NOT NULL,
		food_id INT NOT NULL,
		quantity INT NOT NULL DEFAULT 1,
		price DECIMAL(10,2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
		FOREIGN KEY (food_id) REFERENCES food(id) ON DELETE CASCADE,
		INDEX idx_order_id (order_id),
		INDEX idx_food_id (food_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err = db.Exec(userTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(orderTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(foodTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(orderItemsTable)
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
			INSERT INTO users (name, password, role, address, establishmentName, establishmentAddress) VALUES 
			('cliente1', '123456', 'customer', 'Calle Cliente 123', NULL, NULL),
			('cliente2', '123456', 'customer', 'Calle Cliente 456', NULL, NULL),
			('repartidor1', '123456', 'delivery', NULL, NULL, NULL),
			('repartidor2', '123456', 'delivery', NULL, NULL, NULL),
			('vendedor1', '123456', 'seller', NULL, 'Local Comidas', 'Avenida Principal 123'),
			('vendedor2', '123456', 'seller', NULL, 'Restaurante Delta', 'Calle Secundaria 456')
		`)
		if err != nil {
			return err
		}
	}
	return nil
}
