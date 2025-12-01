package models

// Cart models for Redis session-based storage

type CartItem struct {
	ProductID   string  `json:"product_id" redis:"product_id"`
	SKU         string  `json:"sku" redis:"sku"`
	ProductName string  `json:"product_name" redis:"product_name"`
	Price       float64 `json:"price" redis:"price"`
	Quantity    int     `json:"quantity" redis:"quantity"`
	Subtotal    float64 `json:"subtotal" redis:"subtotal"`
	AddedAt     string  `json:"added_at" redis:"added_at"`
}

type Cart struct {
	SessionID   string               `json:"session_id"`
	Items       map[string]*CartItem `json:"items"` // keyed by SKU
	Subtotal    float64              `json:"subtotal"`
	Tax         float64              `json:"tax"`
	Shipping    float64              `json:"shipping"`
	Total       float64              `json:"total"`
	ItemCount   int                  `json:"item_count"`
	LastUpdated string               `json:"last_updated"`
	ExpiresAt   string               `json:"expires_at"`
}

type AddToCartRequest struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=0"`
}
