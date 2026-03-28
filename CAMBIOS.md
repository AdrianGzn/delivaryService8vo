# Cambios Realizados en Delivery Service API

## 1. Modificaciones en la Base de Datos

### Tabla `users` 
- ✅ Agregados campos `establishmentName` (VARCHAR(255), nullable) y `establishmentAddress` (TEXT, nullable)
- ✅ Nuevo rol: `'seller'` agregado al ENUM de roles (antes solo tenía 'customer' y 'delivery')
- ✅ Los sellers requieren proporcionar `establishmentName` y `establishmentAddress`

### Tabla `orders`
- ✅ Removidos campos `establishmentName` y `establishmentAddress`
- ✅ Agregado campo `seller_id` (INT, FOREIGN KEY) - requerido
- ✅ Los pedidos ahora están asociados a los sellers

### Nueva Tabla `food`
- ✅ Campos: `id`, `seller_id`, `name`, `price`
- ✅ Relación: cada alimento pertenece a un seller

## 2. Cambios en Modelos (models.go)

```go
// User - Estructuras actualizadas
type User struct {
    ID                  int
    Name                string
    Password            string
    Role                string // Ahora: "customer", "delivery", "seller"
    Address             *string
    EstablishmentName   *string // Nuevo campo
    EstablishmentAddr   *string // Nuevo campo
}

// Order - Removidos EstablishmentName/EstablishmentAddr
type Order struct {
    ID         int
    Title      string
    Description string
    Status     string
    Price      float64
    UserID     int
    SellerID   int     // Nuevo campo (requerido)
    DeliveryID *int
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// Food - Nueva estructura
type Food struct {
    ID       int
    SellerID int
    Name     string
    Price    float64
}
```

## 3. Cambio de SSE a WebSocket

### Nuevo: WebSocket Manager (`websocket/websocket_manager.go`)
- ✅ Reemplaza al anterior SSE Manager
- ✅ Gestiona múltiples conexiones WebSocket por usuario
- ✅ Notificaciones en tiempo real para cambios en órdenes

### Nuevo: WebSocket Handler (`handlers/ws_handler.go`)
- ✅ Endpoint: `GET /ws/{userId}` (WebSocket)
- ✅ Endpoint: `GET /ws/status` (consultar usuarios activos)
- ✅ Manejo automático de conexiones y desconexiones

## 4. Actualizaciones en Handlers

### `handlers/user_handlers.go`
- ✅ Validación de nuevo rol `'seller'` en `CreateUser()`
- ✅ Validación: sellers deben proporcionar `establishmentName` y `establishmentAddress`
- ✅ Actualizadas queries SELECT para incluir nuevos campos

### `handlers/order_handlers.go`
- ✅ Cambio de SSEManager a WebSocketManager
- ✅ Actualizado campo `EstablishmentName` → `SellerID` en queries
- ✅ Notificaciones vía WebSocket en lugar de SSE
- ✅ Ahora notifica a: cliente, repartidor y vendedor

### `handlers/login_handlers.go`
- ✅ Actualizado Login para incluir nuevos campos
- ✅ Actualizado Register con validación de sellers
- ✅ Manejo de `establishmentName` y `establishmentAddress`

## 5. Cambios en main.go

### Rutas actualizadas
```
- POST  /login
- POST  /register
- GET   /ws/{userId}          (WebSocket - antes era /sse)
- GET   /ws/status            (Nuevo endpoint)
- GET   /health
- GET   /api/users
- POST  /api/users
- PUT   /api/users/{id}
- DELETE /api/users/{id}
- GET   /api/orders
- POST  /api/orders
- GET   /api/orders/{id}
- GET   /api/orders/user/{userId}
- PATCH /api/orders/{id}/status
- POST  /api/orders/{id}/assign
- DELETE /api/orders/{id}
```

### Cambios de inicialización
- WebSocket Manager reemplaza SSE Manager
- WebSocket Handler incorporado

## 6. Datos de Prueba Actualizados

Se agregaron dos sellers de prueba:
- `vendedor1` con establecimiento "Local Comidas"
- `vendedor2` con establecimiento "Restaurante Delta"

## 7. Dependencias

- ✅ Agregado `github.com/gorilla/websocket v1.5.3` como dependencia directa en go.mod
- Todas las dependencias verificadas

## Notas de Compatibilidad

- El cliente debe cambiar del endpoint `/sse?userId={id}` al endpoint WebSocket `/ws/{userId}`
- Las notificaciones ahora incluyen al vendedor (antes solo cliente y repartidor)
- Se requiere mantener la conexión WebSocket abierta para recibir notificaciones
- Formato de mensaje es idéntico: `{ "event": "...", "data": {...} }`

## Validación

✅ Código compilado correctamente
✅ Todas las funciones de base de datos creadas
✅ Handlers completamente actualizados
✅ Estructura lista para producción
