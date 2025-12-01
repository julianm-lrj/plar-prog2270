package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"julianmorley.ca/con-plar/prog2270/pkg/models"
)

func AddProductsToCache(ctx context.Context, products []*models.Product) error {
	// Cache each product individually using the robust single product caching
	for _, product := range products {
		if err := CacheSingleProduct(ctx, product); err != nil {
			return fmt.Errorf("failed to cache product %s: %w", product.SKU, err)
		}
	}

	return nil
}

func GetProductFromCache(ctx context.Context, productSKU string) (*models.Product, error) {
	client := RedisClient()
	defer client.Close()

	productKey := fmt.Sprintf("product:%s", productSKU)
	productJSON, err := client.Get(ctx, productKey).Result()
	if err != nil {
		return nil, err
	}

	var product models.Product
	if err := json.Unmarshal([]byte(productJSON), &product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal product: %w", err)
	}

	return &product, nil
}

// RemoveProductFromCache removes a product and its related cache entries by SKU
func RemoveProductFromCache(ctx context.Context, product *models.Product) error {
	client := RedisClient()
	defer client.Close()

	// Use pipeline for atomic operations
	pipe := client.TxPipeline()

	// Remove main product cache entry
	productKey := fmt.Sprintf("product:%s", product.SKU)
	pipe.Del(ctx, productKey)

	// Remove SKU mapping
	skuKey := fmt.Sprintf("sku:%s", product.SKU)
	pipe.Del(ctx, skuKey)

	// Remove from category list
	categoryKey := fmt.Sprintf("category:%s", product.Category)
	pipe.LRem(ctx, categoryKey, 0, product.SKU)

	// Remove from recent products list
	pipe.LRem(ctx, "products:recent", 0, product.SKU)

	// Execute all operations
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove product from Redis cache: %w", err)
	}

	return nil
}

// CacheSingleProduct stores a single product in Redis cache using SKU-based keys
func CacheSingleProduct(ctx context.Context, product *models.Product) error {
	client := RedisClient()
	defer client.Close()

	// Serialize product to JSON
	productJSON, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("failed to marshal product %s: %w", product.SKU, err)
	}

	// Use pipeline for atomic operations
	pipe := client.TxPipeline()

	// Store individual product with key pattern: product:{sku}
	productKey := fmt.Sprintf("product:%s", product.SKU)
	pipe.Set(ctx, productKey, productJSON, 24*time.Hour)

	// Store product SKU mapping for quick lookups: sku:{sku} -> {sku} (for consistency)
	skuKey := fmt.Sprintf("sku:%s", product.SKU)
	pipe.Set(ctx, skuKey, product.SKU, 24*time.Hour)

	// Add to category-based lists for filtering
	categoryKey := fmt.Sprintf("category:%s", product.Category)
	pipe.LPush(ctx, categoryKey, product.SKU)
	pipe.Expire(ctx, categoryKey, 24*time.Hour)

	// Add to recent products list
	pipe.LPush(ctx, "products:recent", product.SKU)
	// Keep only the 100 most recent products
	pipe.LTrim(ctx, "products:recent", 0, 99)
	pipe.Expire(ctx, "products:recent", 24*time.Hour)

	// Execute all operations atomically
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute Redis pipeline for product %s: %w", product.SKU, err)
	}

	return nil
}

func GetProductBySKUFromCache(ctx context.Context, sku string) (*models.Product, error) {
	client := RedisClient()
	defer client.Close()

	skuKey := fmt.Sprintf("sku:%s", sku)
	productID, err := client.Get(ctx, skuKey).Result()
	if err != nil {
		return nil, err
	}

	return GetProductFromCache(ctx, productID)
}

// Cart operations using Redis Hashes

// GetCart retrieves a cart by session ID
func GetCart(ctx context.Context, sessionID string) (*models.Cart, error) {
	client := RedisClient()
	defer client.Close()

	cartKey := fmt.Sprintf("cart:%s", sessionID)

	// Check if cart exists
	exists, err := client.Exists(ctx, cartKey).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		// Return empty cart
		return createEmptyCart(sessionID), nil
	}

	// Get all cart data
	cartData, err := client.HGetAll(ctx, cartKey).Result()
	if err != nil {
		return nil, err
	}

	// Parse items from individual item keys
	itemPattern := fmt.Sprintf("cart:%s:item:*", sessionID)
	itemKeys, err := client.Keys(ctx, itemPattern).Result()
	if err != nil {
		return nil, err
	}

	items := make(map[string]*models.CartItem)
	for _, itemKey := range itemKeys {
		itemData, err := client.HGetAll(ctx, itemKey).Result()
		if err != nil {
			continue
		}

		item := &models.CartItem{}
		if productID, ok := itemData["product_id"]; ok {
			item.ProductID = productID
		}
		if sku, ok := itemData["sku"]; ok {
			item.SKU = sku
		}
		if name, ok := itemData["product_name"]; ok {
			item.ProductName = name
		}
		if priceStr, ok := itemData["price"]; ok {
			if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
				item.Price = price
			}
		}
		if qtyStr, ok := itemData["quantity"]; ok {
			if qty, err := strconv.Atoi(qtyStr); err == nil {
				item.Quantity = qty
			}
		}
		if subtotalStr, ok := itemData["subtotal"]; ok {
			if subtotal, err := strconv.ParseFloat(subtotalStr, 64); err == nil {
				item.Subtotal = subtotal
			}
		}
		if addedAt, ok := itemData["added_at"]; ok {
			item.AddedAt = addedAt
		}

		items[item.SKU] = item // Key by SKU instead of ProductID
	}

	cart := &models.Cart{
		SessionID: sessionID,
		Items:     items,
	}

	// Parse cart metadata
	if subtotalStr, ok := cartData["subtotal"]; ok {
		if subtotal, err := strconv.ParseFloat(subtotalStr, 64); err == nil {
			cart.Subtotal = subtotal
		}
	}
	if taxStr, ok := cartData["tax"]; ok {
		if tax, err := strconv.ParseFloat(taxStr, 64); err == nil {
			cart.Tax = tax
		}
	}
	if shippingStr, ok := cartData["shipping"]; ok {
		if shipping, err := strconv.ParseFloat(shippingStr, 64); err == nil {
			cart.Shipping = shipping
		}
	}
	if totalStr, ok := cartData["total"]; ok {
		if total, err := strconv.ParseFloat(totalStr, 64); err == nil {
			cart.Total = total
		}
	}
	if itemCountStr, ok := cartData["item_count"]; ok {
		if itemCount, err := strconv.Atoi(itemCountStr); err == nil {
			cart.ItemCount = itemCount
		}
	}
	if lastUpdated, ok := cartData["last_updated"]; ok {
		cart.LastUpdated = lastUpdated
	}
	if expiresAt, ok := cartData["expires_at"]; ok {
		cart.ExpiresAt = expiresAt
	}

	return cart, nil
}

// AddToCart adds an item to the cart
func AddToCart(ctx context.Context, sessionID, sku string, quantity int, product *models.Product) (*models.Cart, error) {
	client := RedisClient()
	defer client.Close()

	// Get existing cart
	cart, err := GetCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Create or update cart item
	now := time.Now().UTC().Format(time.RFC3339)
	subtotal := float64(quantity) * product.Price

	if existingItem, exists := cart.Items[sku]; exists {
		// Update existing item
		existingItem.Quantity += quantity
		existingItem.Subtotal = float64(existingItem.Quantity) * existingItem.Price
	} else {
		// Add new item
		cart.Items[sku] = &models.CartItem{
			ProductID:   product.ID.Hex(),
			SKU:         product.SKU,
			ProductName: product.Name,
			Price:       product.Price,
			Quantity:    quantity,
			Subtotal:    subtotal,
			AddedAt:     now,
		}
	}

	// Recalculate cart totals
	calculateCartTotals(cart)
	cart.LastUpdated = now
	cart.ExpiresAt = time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)

	// Save to Redis
	err = saveCartToRedis(ctx, client, cart)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

// UpdateCartItem updates the quantity of an item in the cart
func UpdateCartItem(ctx context.Context, sessionID, sku string, quantity int) (*models.Cart, error) {
	client := RedisClient()
	defer client.Close()

	// Get existing cart
	cart, err := GetCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Check if item exists
	item, exists := cart.Items[sku]
	if !exists {
		return nil, fmt.Errorf("item not found in cart")
	}

	if quantity == 0 {
		// Remove item from cart
		delete(cart.Items, sku)
		// Remove from Redis
		itemKey := fmt.Sprintf("cart:%s:item:%s", sessionID, sku)
		client.Del(ctx, itemKey)
	} else {
		// Update quantity
		item.Quantity = quantity
		item.Subtotal = float64(quantity) * item.Price
	}

	// Recalculate cart totals
	calculateCartTotals(cart)
	cart.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	cart.ExpiresAt = time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)

	// Save to Redis
	err = saveCartToRedis(ctx, client, cart)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

// RemoveFromCart removes an item from the cart
func RemoveFromCart(ctx context.Context, sessionID, sku string) (*models.Cart, error) {
	return UpdateCartItem(ctx, sessionID, sku, 0)
}

// ClearCart removes all items from the cart
func ClearCart(ctx context.Context, sessionID string) error {
	client := RedisClient()
	defer client.Close()

	// Remove all cart-related keys
	cartPattern := fmt.Sprintf("cart:%s*", sessionID)
	keys, err := client.Keys(ctx, cartPattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return client.Del(ctx, keys...).Err()
	}

	return nil
}

// Helper functions

func createEmptyCart(sessionID string) *models.Cart {
	now := time.Now().UTC().Format(time.RFC3339)
	return &models.Cart{
		SessionID:   sessionID,
		Items:       make(map[string]*models.CartItem),
		Subtotal:    0,
		Tax:         0,
		Shipping:    0,
		Total:       0,
		ItemCount:   0,
		LastUpdated: now,
		ExpiresAt:   time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
	}
}

func calculateCartTotals(cart *models.Cart) {
	cart.Subtotal = 0
	cart.ItemCount = 0

	for _, item := range cart.Items {
		cart.Subtotal += item.Subtotal
		cart.ItemCount += item.Quantity
	}

	// Calculate tax (assuming 10% tax rate)
	cart.Tax = cart.Subtotal * 0.10

	// Calculate shipping (free shipping over $50, otherwise $5.99)
	cart.Shipping = 0
	if cart.Subtotal > 0 && cart.Subtotal < 50 {
		cart.Shipping = 5.99
	}

	// Calculate total
	cart.Total = cart.Subtotal + cart.Tax + cart.Shipping
}

func saveCartToRedis(ctx context.Context, client *redisclient.Client, cart *models.Cart) error {
	cartKey := fmt.Sprintf("cart:%s", cart.SessionID)

	// Save cart metadata
	cartData := map[string]interface{}{
		"subtotal":     fmt.Sprintf("%.2f", cart.Subtotal),
		"tax":          fmt.Sprintf("%.2f", cart.Tax),
		"shipping":     fmt.Sprintf("%.2f", cart.Shipping),
		"total":        fmt.Sprintf("%.2f", cart.Total),
		"item_count":   fmt.Sprintf("%d", cart.ItemCount),
		"last_updated": cart.LastUpdated,
		"expires_at":   cart.ExpiresAt,
	}

	err := client.HSet(ctx, cartKey, cartData).Err()
	if err != nil {
		return err
	}

	// Set TTL for cart (1 hour)
	client.Expire(ctx, cartKey, 1*time.Hour)

	// Save individual items
	for sku, item := range cart.Items {
		itemKey := fmt.Sprintf("cart:%s:item:%s", cart.SessionID, sku)
		itemData := map[string]interface{}{
			"product_id":   item.ProductID,
			"sku":          item.SKU,
			"product_name": item.ProductName,
			"price":        fmt.Sprintf("%.2f", item.Price),
			"quantity":     fmt.Sprintf("%d", item.Quantity),
			"subtotal":     fmt.Sprintf("%.2f", item.Subtotal),
			"added_at":     item.AddedAt,
		}

		err := client.HSet(ctx, itemKey, itemData).Err()
		if err != nil {
			return err
		}

		// Set TTL for item (1 hour)
		client.Expire(ctx, itemKey, 1*time.Hour)
	}

	return nil
}
