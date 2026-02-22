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
	ID                int     `json:"id"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	Status            string  `json:"status"` // "pending", "pickup", "in_coming", "arrived", "delivered"
	EstablishmentName string  `json:"establishmentName"`
	EstablishmentAddr string  `json:"establishmentAddress"`
	Price             float64 `json:"price"`
	UserID            int     `json:"userId"`
	DeliveryID        *int    `json:"deliveryId,omitempty"`
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
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('customer', 'delivery')),
		address TEXT
	);`

	orderTable := `
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('pending', 'pickup', 'in_coming', 'arrived', 'delivered')),
		establishmentName TEXT NOT NULL,
		establishmentAddress TEXT NOT NULL,
		price REAL NOT NULL,
		user_id INTEGER NOT NULL,
		delivery_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (delivery_id) REFERENCES users(id)
	);`

	_, err := db.Exec(userTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(orderTable)
	return err
}