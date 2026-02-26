package handlers

import (
    "database/sql"
    "encoding/json"
    "io"
    "bytes"
    "log"
    "net/http"

    "deliveryService/models"
)

type LoginHandler struct {
    DB *sql.DB
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
    var loginReq models.LoginRequest
    err := json.NewDecoder(r.Body).Decode(&loginReq)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var user models.User
    err = h.DB.QueryRow(
        "SELECT id, name, password, role, address FROM users WHERE name = ?",
        loginReq.Name,
    ).Scan(&user.ID, &user.Name, &user.Password, &user.Role, &user.Address)

    if err == sql.ErrNoRows {
        http.Error(w, "Usuario no encontrado", http.StatusUnauthorized)
        return
    } else if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if user.Password != loginReq.Password {
        http.Error(w, "Contraseña incorrecta", http.StatusUnauthorized)
        return
    }

    user.Password = ""

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func (h *LoginHandler) Register(w http.ResponseWriter, r *http.Request) {
    // Verificar Content-Type
    if r.Header.Get("Content-Type") != "application/json" {
        http.Error(w, "Content-Type debe ser application/json", http.StatusBadRequest)
        return
    }

    // Leer todo el body para depuración
    body, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error leyendo body: %v", err)
        http.Error(w, "Error leyendo body: " + err.Error(), http.StatusBadRequest)
        return
    }
    
    // Imprimir el body recibido
    log.Printf("=== REGISTER REQUEST ===")
    log.Printf("Body recibido (raw): %s", string(body))
    
    // Restaurar el body para poder decodificarlo
    r.Body = io.NopCloser(bytes.NewBuffer(body))
    
    // Intentar decodificar a un mapa primero
    var data map[string]interface{}
    err = json.NewDecoder(r.Body).Decode(&data)
    if err != nil {
        log.Printf("Error decodificando JSON: %v", err)
        http.Error(w, "Error decodificando JSON: " + err.Error(), http.StatusBadRequest)
        return
    }
    
    log.Printf("Datos decodificados (map): %+v", data)
    
    // Extraer valores manualmente
    name, _ := data["name"].(string)
    password, _ := data["password"].(string)
    role, _ := data["role"].(string)
    address, _ := data["address"].(string)
    
    log.Printf("Valores extraídos - Name: '%s', Password: '%s', Role: '%s', Address: '%s'", 
        name, password, role, address)
    
    // Validar campos requeridos
    if name == "" || password == "" {
        log.Printf("ERROR: Campos requeridos vacíos")
        http.Error(w, "Nombre y contraseña son requeridos", http.StatusBadRequest)
        return
    }
    
    // Validar rol
    if role != "customer" && role != "delivery" {
        http.Error(w, "Rol inválido. Debe ser 'customer' o 'delivery'", http.StatusBadRequest)
        return
    }
    
    log.Printf("Intentando insertar en BD: %s, %s, %s, %s", name, password, role, address)
    
    // Insertar en BD
    result, err := h.DB.Exec(
        "INSERT INTO users (name, password, role, address) VALUES (?, ?, ?, ?)",
        name, password, role, address,
    )
    if err != nil {
        log.Printf("Error SQL: %v", err)
        http.Error(w, "Error SQL: " + err.Error(), http.StatusInternalServerError)
        return
    }
    
    id, err := result.LastInsertId()
    if err != nil {
        log.Printf("Error obteniendo LastInsertId: %v", err)
        http.Error(w, "Error obteniendo ID: " + err.Error(), http.StatusInternalServerError)
        return
    }
    
    log.Printf("Usuario creado con ID: %d", id)
    
    // Crear respuesta
    response := models.User{
        ID:      int(id),
        Name:    name,
        Role:    role,
        Address: &address,
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(response)
    
    log.Printf("=== REGISTER COMPLETADO ===")
}