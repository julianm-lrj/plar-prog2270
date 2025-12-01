package router

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
	"julianmorley.ca/con-plar/prog2270/pkg/ai"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
	"julianmorley.ca/con-plar/prog2270/pkg/models"
	"julianmorley.ca/con-plar/prog2270/pkg/mongo"
	"julianmorley.ca/con-plar/prog2270/pkg/redis"
)

func HealthCheck(c *gin.Context) {
	db := mongo.GetDatabase()
	if err := db.Client().Ping(c, nil); err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Database connection failed", nil))
		return
	}
	c.JSON(http.StatusOK, global.SuccessResponse(map[string]string{"status": "OK", "database": "Connected"}))
}

func GetAllProducts(c *gin.Context) {
	products, err := mongo.GetAllProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to get products", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(products))
}

// GetProductBySKU retrieves a product by SKU with Redis caching
func GetProductBySKU(c *gin.Context) {
	sku := c.Param("sku") // Parameter is named 'sku'

	// Validate SKU format (basic validation)
	if len(sku) < 3 || len(sku) > 50 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid SKU format", []global.ValidationError{
			{Field: "sku", Message: "SKU must be between 3 and 50 characters", Code: "invalid_format"},
		}))
		return
	}

	ctx := c.Request.Context()

	// Try Redis cache first using SKU
	product, err := redis.GetProductBySKUFromCache(ctx, sku)
	if err == nil {
		// Found in cache, return immediately
		c.Header("X-Cache", "HIT")
		c.JSON(http.StatusOK, global.SuccessResponse(product))
		return
	}

	// Cache miss, check MongoDB by SKU
	product, err = mongo.GetProductBySKU(ctx, sku)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "mongo: no documents in result" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Product not found", []global.ValidationError{
				{Field: "sku", Message: "No product exists with this SKU", Code: "not_found"},
			}))
			return
		}
		// Other database error
		log.Printf("Error fetching product from MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to fetch product", nil))
		return
	}

	// Found in MongoDB, cache it for future requests
	if cacheErr := redis.CacheSingleProduct(ctx, product); cacheErr != nil {
		// Log cache error but don't fail the request
		log.Printf("Warning: Failed to cache product in Redis: %v", cacheErr)
	}

	// Return product with cache miss indicator
	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, global.SuccessResponse(product))
}

// EditProductBySKU updates specific fields of a product by SKU
func EditProductBySKU(c *gin.Context) {
	sku := c.Param("sku")

	// Validate SKU format
	if len(sku) < 3 || len(sku) > 50 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid SKU format", []global.ValidationError{
			{Field: "sku", Message: "SKU must be between 3 and 50 characters", Code: "invalid_format"},
		}))
		return
	}

	ctx := c.Request.Context()

	// Parse JSON body into a map for partial updates
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid JSON format", []global.ValidationError{
			{Field: "body", Message: err.Error(), Code: "json_parse_error"},
		}))
		return
	}

	// Validate that we have updates to apply
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No updates provided", []global.ValidationError{
			{Field: "body", Message: "Request body must contain at least one field to update", Code: "empty_updates"},
		}))
		return
	}

	// Prevent updating immutable fields - remove them from updates instead of erroring
	immutableFields := []string{"_id", "id", "sku", "created_at"}
	for _, field := range immutableFields {
		if _, exists := updates[field]; exists {
			delete(updates, field)
			log.Printf("Warning: Removed immutable field '%s' from update request", field)
		}
	}

	// Check if we still have updates after removing immutable fields
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No valid updates provided", []global.ValidationError{
			{Field: "body", Message: "All provided fields are immutable and cannot be updated", Code: "no_valid_updates"},
		}))
		return
	}

	// Update the product in MongoDB
	updatedProduct, err := mongo.UpdateProductBySKU(ctx, sku, updates)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "mongo: no documents in result" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Product not found", []global.ValidationError{
				{Field: "sku", Message: "No product exists with this SKU", Code: "not_found"},
			}))
			return
		}
		// Other database error
		log.Printf("Error updating product in MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to update product", nil))
		return
	}

	// Update the entire document in Redis cache
	if cacheErr := redis.CacheSingleProduct(ctx, updatedProduct); cacheErr != nil {
		// Log cache error but don't fail the request since DB update succeeded
		log.Printf("Warning: Failed to update product cache in Redis: %v", cacheErr)
	}

	// Return the updated product
	c.Header("X-Cache", "REFRESHED")
	c.JSON(http.StatusOK, global.SuccessResponse(updatedProduct))
}

// DeleteProductBySKU deletes a product by SKU from both database and cache
func DeleteProductBySKU(c *gin.Context) {
	sku := c.Param("sku")

	// Validate SKU format
	if len(sku) < 3 || len(sku) > 50 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid SKU format", []global.ValidationError{
			{Field: "sku", Message: "SKU must be between 3 and 50 characters", Code: "invalid_format"},
		}))
		return
	}

	ctx := c.Request.Context()

	// Delete the product from MongoDB (this also returns the deleted product for cache cleanup)
	deletedProduct, err := mongo.DeleteProductBySKU(ctx, sku)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "mongo: no documents in result" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Product not found", []global.ValidationError{
				{Field: "sku", Message: "No product exists with this SKU", Code: "not_found"},
			}))
			return
		}
		// Other database error
		log.Printf("Error deleting product from MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to delete product", nil))
		return
	}

	// Remove from Redis cache
	if cacheErr := redis.RemoveProductFromCache(ctx, deletedProduct); cacheErr != nil {
		// Log cache error but don't fail the request since DB deletion succeeded
		log.Printf("Warning: Failed to remove product from Redis cache: %v", cacheErr)
	}

	// Return success with the deleted product info
	c.Header("X-Cache", "DELETED")
	c.JSON(http.StatusOK, global.SuccessResponse(map[string]interface{}{
		"deleted_sku": deletedProduct.SKU,
		"message":     "Product successfully deleted",
	}))
}

func CreateNewProducts(c *gin.Context) {
	var req []models.CreateProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid request data", []global.ValidationError{
			{Field: "request", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	if len(req) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No products provided", []global.ValidationError{
			{Field: "products", Message: "At least one product is required", Code: "empty_array"},
		}))
		return
	}

	products := make([]*models.Product, len(req))
	for i, productReq := range req {
		products[i] = productReq.ToProduct()
	}

	createdProducts, err := mongo.CreateProducts(c.Request.Context(), products)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to create products", nil))
		return
	}

	if err := redis.AddProductsToCache(c.Request.Context(), createdProducts); err != nil {
		// Log the error but don't fail the request since MongoDB succeeded
		// In production, you might want to use a proper logger here
		log.Printf("Warning: Failed to cache products in Redis: %v", err)
	}

	c.JSON(http.StatusCreated, global.SuccessResponse(map[string]interface{}{
		"products": createdProducts,
		"count":    len(createdProducts),
	}))
}

// BulkEditProducts updates multiple products by their SKUs
func BulkEditProducts(c *gin.Context) {
	var bulkUpdates []map[string]interface{}

	if err := c.ShouldBindJSON(&bulkUpdates); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid JSON format", []global.ValidationError{
			{Field: "body", Message: err.Error(), Code: "json_parse_error"},
		}))
		return
	}

	// Validate that we have updates to apply
	if len(bulkUpdates) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No updates provided", []global.ValidationError{
			{Field: "body", Message: "Request body must contain at least one product update", Code: "empty_updates"},
		}))
		return
	}

	ctx := c.Request.Context()
	var updatedProducts []*models.Product
	var errors []global.ValidationError

	// Process each product update
	for i, updateData := range bulkUpdates {
		// Extract SKU from the update data
		skuInterface, exists := updateData["sku"]
		if !exists {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].sku", i),
				Message: "SKU is required for each product update",
				Code:    "missing_sku",
			})
			continue
		}

		sku, ok := skuInterface.(string)
		if !ok || len(sku) < 3 || len(sku) > 50 {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].sku", i),
				Message: "SKU must be a string between 3 and 50 characters",
				Code:    "invalid_sku_format",
			})
			continue
		}

		// Remove SKU from updates map since it's immutable
		updates := make(map[string]interface{})
		for key, value := range updateData {
			if key != "sku" {
				updates[key] = value
			}
		}

		// Remove immutable fields (same logic as EditProductBySKU)
		immutableFields := []string{"_id", "id", "sku", "created_at"}
		for _, field := range immutableFields {
			if _, exists := updates[field]; exists {
				delete(updates, field)
				log.Printf("Warning: Removed immutable field '%s' from bulk update for SKU %s", field, sku)
			}
		}

		// Skip if no valid updates remain
		if len(updates) == 0 {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d]", i),
				Message: fmt.Sprintf("No valid fields to update for SKU %s", sku),
				Code:    "no_valid_updates",
			})
			continue
		}

		// Update the product in MongoDB
		updatedProduct, err := mongo.UpdateProductBySKU(ctx, sku, updates)
		if err != nil {
			// Handle product not found
			if err.Error() == "mongo: no documents in result" {
				errors = append(errors, global.ValidationError{
					Field:   fmt.Sprintf("[%d].sku", i),
					Message: fmt.Sprintf("No product exists with SKU %s", sku),
					Code:    "not_found",
				})
				continue
			}
			// Handle other database errors
			log.Printf("Error updating product %s in MongoDB: %v", sku, err)
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].sku", i),
				Message: fmt.Sprintf("Failed to update product with SKU %s", sku),
				Code:    "update_failed",
			})
			continue
		}

		// Update Redis cache
		if cacheErr := redis.CacheSingleProduct(ctx, updatedProduct); cacheErr != nil {
			log.Printf("Warning: Failed to update product cache in Redis for SKU %s: %v", sku, cacheErr)
		}

		updatedProducts = append(updatedProducts, updatedProduct)
	}

	// Determine response status
	statusCode := http.StatusOK
	if len(updatedProducts) == 0 {
		// All updates failed
		statusCode = http.StatusBadRequest
	} else if len(errors) > 0 {
		// Partial success
		statusCode = http.StatusMultiStatus
	}

	// Return response with results and any errors
	responseData := map[string]interface{}{
		"updated_products": updatedProducts,
		"success_count":    len(updatedProducts),
		"total_requested":  len(bulkUpdates),
	}

	if len(errors) > 0 {
		responseData["errors"] = errors
		responseData["error_count"] = len(errors)
	}

	c.Header("X-Cache", "BULK-REFRESHED")
	c.JSON(statusCode, global.SuccessResponse(responseData))
}

// BulkDeleteRequest represents the structure for bulk delete requests
type BulkDeleteRequest struct {
	SKU string `json:"sku" binding:"required"`
}

// BulkDeleteProducts deletes multiple products by SKU array
func BulkDeleteProducts(c *gin.Context) {
	var deleteRequests []BulkDeleteRequest

	// Parse JSON body - expecting array of objects with sku property
	if err := c.ShouldBindJSON(&deleteRequests); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid JSON format", []global.ValidationError{
			{Field: "body", Message: "Expected array of objects with 'sku' property", Code: "json_parse_error"},
		}))
		return
	}

	// Validate that we have SKUs to delete
	if len(deleteRequests) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No SKUs provided", []global.ValidationError{
			{Field: "body", Message: "Request body must contain at least one object with SKU to delete", Code: "empty_array"},
		}))
		return
	}

	ctx := c.Request.Context()
	var deletedProducts []*models.Product
	var errors []global.ValidationError
	successCount := 0

	// Process each SKU for deletion
	for i, deleteReq := range deleteRequests {
		sku := deleteReq.SKU

		// Validate SKU format
		if len(sku) < 3 || len(sku) > 50 {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].sku", i),
				Message: "SKU must be between 3 and 50 characters",
				Code:    "invalid_format",
			})
			continue
		}

		// Delete the product from MongoDB
		deletedProduct, err := mongo.DeleteProductBySKU(ctx, sku)
		if err != nil {
			// Handle not found error
			if err.Error() == "mongo: no documents in result" {
				errors = append(errors, global.ValidationError{
					Field:   fmt.Sprintf("[%d].sku", i),
					Message: fmt.Sprintf("No product exists with SKU %s", sku),
					Code:    "not_found",
				})
			} else {
				// Other database error
				log.Printf("Error deleting product %s from MongoDB: %v", sku, err)
				errors = append(errors, global.ValidationError{
					Field:   fmt.Sprintf("[%d].sku", i),
					Message: "Database error occurred",
					Code:    "database_error",
				})
			}
			continue
		}

		// Remove from Redis cache
		if cacheErr := redis.RemoveProductFromCache(ctx, deletedProduct); cacheErr != nil {
			// Log cache error but don't fail the request since DB deletion succeeded
			log.Printf("Warning: Failed to remove product %s from Redis cache: %v", sku, cacheErr)
		}

		deletedProducts = append(deletedProducts, deletedProduct)
		successCount++
	}

	// Prepare response data with SKU array instead of full objects
	deletedSKUs := make([]string, 0, len(deletedProducts))
	for _, product := range deletedProducts {
		deletedSKUs = append(deletedSKUs, product.SKU)
	}

	responseData := map[string]interface{}{
		"deleted_products": deletedSKUs,
		"success_count":    successCount,
		"total_requested":  len(deleteRequests),
	}

	// Add error information if any
	if len(errors) > 0 {
		responseData["error_count"] = len(errors)
		responseData["errors"] = errors
	}

	// Determine status code based on results
	statusCode := http.StatusOK
	if successCount == 0 {
		// All deletions failed
		statusCode = http.StatusBadRequest
	} else if len(errors) > 0 {
		// Partial success
		statusCode = http.StatusMultiStatus
	}

	// Return response
	c.Header("X-Cache", "BULK-DELETED")
	c.JSON(statusCode, global.SuccessResponse(responseData))
}

func GetAllOrders(c *gin.Context) {
	orders, err := mongo.GetAllOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to get orders", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(orders))
}

// CreateNewOrders creates multiple orders from an array of order requests
func CreateNewOrders(c *gin.Context) {
	var orderRequests []models.CreateOrderRequest

	if err := c.ShouldBindJSON(&orderRequests); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid JSON format", []global.ValidationError{
			{Field: "body", Message: err.Error(), Code: "json_parse_error"},
		}))
		return
	}

	// Validate that we have orders to create
	if len(orderRequests) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No orders provided", []global.ValidationError{
			{Field: "orders", Message: "At least one order is required", Code: "empty_array"},
		}))
		return
	}

	ctx := c.Request.Context()

	// Create orders using the bulk creation helper
	createdOrders, errors := mongo.CreateNewOrders(ctx, orderRequests)

	// Check if all orders failed
	allFailed := len(errors) > 0
	for _, err := range errors {
		if err == nil {
			allFailed = false
			break
		}
	}

	if allFailed {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to create any orders", []global.ValidationError{
			{Field: "orders", Message: "All order creation attempts failed", Code: "creation_failed"},
		}))
		return
	}

	// Prepare response data
	var successfulOrders []models.Order
	var failedOrders []map[string]interface{}

	for i, order := range createdOrders {
		if i < len(errors) && errors[i] != nil {
			failedOrders = append(failedOrders, map[string]interface{}{
				"index": i,
				"error": errors[i].Error(),
				"order": orderRequests[i],
			})
		} else {
			successfulOrders = append(successfulOrders, order)
		}
	}

	responseData := map[string]interface{}{
		"orders":         successfulOrders,
		"total_created":  len(successfulOrders),
		"total_failed":   len(failedOrders),
		"total_attempts": len(orderRequests),
	}

	// Add failed orders to response if any
	if len(failedOrders) > 0 {
		responseData["failed_orders"] = failedOrders
	}

	// Determine status code
	statusCode := http.StatusCreated
	if len(failedOrders) > 0 {
		statusCode = http.StatusMultiStatus // 207 for partial success
	}

	c.JSON(statusCode, global.SuccessResponse(responseData))
}

// BulkEditOrders updates multiple orders by their order numbers
func BulkEditOrders(c *gin.Context) {
	var bulkUpdates []map[string]interface{}

	if err := c.ShouldBindJSON(&bulkUpdates); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid JSON format", []global.ValidationError{
			{Field: "body", Message: err.Error(), Code: "json_parse_error"},
		}))
		return
	}

	// Validate that we have updates to apply
	if len(bulkUpdates) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No updates provided", []global.ValidationError{
			{Field: "body", Message: "Request body must contain at least one order update", Code: "empty_updates"},
		}))
		return
	}

	ctx := c.Request.Context()
	var updatedOrders []*models.Order
	var errors []global.ValidationError

	// Process each order update
	for i, updateData := range bulkUpdates {
		// Extract order_number from the update data
		orderNumberInterface, exists := updateData["order_number"]
		if !exists {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].order_number", i),
				Message: "Order number is required for each order update",
				Code:    "missing_order_number",
			})
			continue
		}

		orderNumber, ok := orderNumberInterface.(string)
		if !ok || len(orderNumber) < 3 || len(orderNumber) > 100 {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].order_number", i),
				Message: "Order number must be a string between 3 and 100 characters",
				Code:    "invalid_order_number_format",
			})
			continue
		}

		updates := make(map[string]interface{})
		for key, value := range updateData {
			updates[key] = value
		}

		// Remove immutable fields
		immutableFields := []string{"_id", "id", "order_number", "created_at", "customer_id", "customer_email"}
		for _, field := range immutableFields {
			if _, exists := updates[field]; exists {
				delete(updates, field)
				log.Printf("Warning: Removed immutable field '%s' from bulk update for order %s", field, orderNumber)
			}
		}

		// Skip if no valid updates remain
		if len(updates) == 0 {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d]", i),
				Message: fmt.Sprintf("No valid fields to update for order %s", orderNumber),
				Code:    "no_valid_updates",
			})
			continue
		}

		// Update the order in MongoDB
		updatedOrder, err := mongo.UpdateOrderByNumber(ctx, orderNumber, updates)
		if err != nil {
			// Handle order not found
			if err.Error() == "mongo: no documents in result" {
				errors = append(errors, global.ValidationError{
					Field:   fmt.Sprintf("[%d].order_number", i),
					Message: fmt.Sprintf("No order exists with order number %s", orderNumber),
					Code:    "not_found",
				})
				continue
			}
			// Handle other database errors
			log.Printf("Error updating order %s in MongoDB: %v", orderNumber, err)
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].order_number", i),
				Message: fmt.Sprintf("Failed to update order with order number %s", orderNumber),
				Code:    "update_failed",
			})
			continue
		}

		updatedOrders = append(updatedOrders, updatedOrder)
	}

	// Determine response status
	statusCode := http.StatusOK
	if len(updatedOrders) == 0 {
		// All updates failed
		statusCode = http.StatusBadRequest
	} else if len(errors) > 0 {
		// Partial success
		statusCode = http.StatusMultiStatus
	}

	// Return response with results and any errors
	responseData := map[string]interface{}{
		"updated_orders":  updatedOrders,
		"success_count":   len(updatedOrders),
		"total_requested": len(bulkUpdates),
	}

	if len(errors) > 0 {
		responseData["errors"] = errors
		responseData["error_count"] = len(errors)
	}

	c.Header("X-Cache", "BULK-UPDATED")
	c.JSON(statusCode, global.SuccessResponse(responseData))
}

// BulkDeleteOrderRequest represents the structure for bulk order delete requests
type BulkDeleteOrderRequest struct {
	OrderNumber string `json:"order_number" binding:"required"`
}

// BulkDeleteOrders deletes multiple orders by order number array
func BulkDeleteOrders(c *gin.Context) {
	var deleteRequests []BulkDeleteOrderRequest

	// Parse JSON body - expecting array of objects with order_number property
	if err := c.ShouldBindJSON(&deleteRequests); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid JSON format", []global.ValidationError{
			{Field: "body", Message: "Expected array of objects with 'order_number' property", Code: "json_parse_error"},
		}))
		return
	}

	// Validate that we have order numbers to delete
	if len(deleteRequests) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No order numbers provided", []global.ValidationError{
			{Field: "body", Message: "Request body must contain at least one order to delete", Code: "empty_array"},
		}))
		return
	}

	ctx := c.Request.Context()
	var deletedOrders []*models.Order
	var errors []global.ValidationError
	successCount := 0

	// Process each order number for deletion
	for i, deleteReq := range deleteRequests {
		orderNumber := deleteReq.OrderNumber

		// Validate order number format
		if len(orderNumber) < 3 || len(orderNumber) > 100 {
			errors = append(errors, global.ValidationError{
				Field:   fmt.Sprintf("[%d].order_number", i),
				Message: "Order number must be between 3 and 100 characters",
				Code:    "invalid_format",
			})
			continue
		}

		// Delete the order from MongoDB
		deletedOrder, err := mongo.DeleteOrderByNumber(ctx, orderNumber)
		if err != nil {
			// Handle not found error
			if err.Error() == "mongo: no documents in result" || err.Error() == "order not found" {
				errors = append(errors, global.ValidationError{
					Field:   fmt.Sprintf("[%d].order_number", i),
					Message: fmt.Sprintf("No order exists with order number %s", orderNumber),
					Code:    "not_found",
				})
			} else {
				// Other database error
				log.Printf("Error deleting order %s from MongoDB: %v", orderNumber, err)
				errors = append(errors, global.ValidationError{
					Field:   fmt.Sprintf("[%d].order_number", i),
					Message: "Database error occurred",
					Code:    "database_error",
				})
			}
			continue
		}

		deletedOrders = append(deletedOrders, deletedOrder)
		successCount++
	}

	// Prepare response data with order number array instead of full objects
	deletedOrderNumbers := make([]string, 0, len(deletedOrders))
	for _, order := range deletedOrders {
		deletedOrderNumbers = append(deletedOrderNumbers, order.OrderNumber)
	}

	responseData := map[string]interface{}{
		"deleted_orders":  deletedOrderNumbers,
		"success_count":   successCount,
		"total_requested": len(deleteRequests),
	}

	// Add error information if any
	if len(errors) > 0 {
		responseData["error_count"] = len(errors)
		responseData["errors"] = errors
	}

	// Determine status code based on results
	statusCode := http.StatusOK
	if successCount == 0 {
		// All deletions failed
		statusCode = http.StatusBadRequest
	} else if len(errors) > 0 {
		// Partial success
		statusCode = http.StatusMultiStatus
	}

	// Return response
	c.Header("X-Cache", "BULK-DELETED")
	c.JSON(statusCode, global.SuccessResponse(responseData))
}

// GetOrderByNumber retrieves a single order by its order number
func GetOrderByNumber(c *gin.Context) {
	orderNumber := c.Param("orderNumber") // Parameter is named 'orderNumber'

	// Validate order number format (basic validation)
	if len(orderNumber) < 3 || len(orderNumber) > 100 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid order number format", []global.ValidationError{
			{Field: "order_number", Message: "Order number must be between 3 and 100 characters", Code: "invalid_format"},
		}))
		return
	}

	ctx := c.Request.Context()

	// Fetch order from MongoDB by order number
	order, err := mongo.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "mongo: no documents in result" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Order not found", []global.ValidationError{
				{Field: "order_number", Message: "No order exists with this order number", Code: "not_found"},
			}))
			return
		}
		// Other database error
		log.Printf("Error fetching order from MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to fetch order", nil))
		return
	}

	// Return order
	c.JSON(http.StatusOK, global.SuccessResponse(order))
}

// EditOrderByNumber updates specific fields of an order by order number
func EditOrderByNumber(c *gin.Context) {
	orderNumber := c.Param("orderNumber")

	// Validate order number format
	if len(orderNumber) < 3 || len(orderNumber) > 100 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid order number format", []global.ValidationError{
			{Field: "order_number", Message: "Order number must be between 3 and 100 characters", Code: "invalid_format"},
		}))
		return
	}

	ctx := c.Request.Context()

	// Parse JSON body into a map for partial updates
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid JSON format", []global.ValidationError{
			{Field: "body", Message: err.Error(), Code: "json_parse_error"},
		}))
		return
	}

	// Validate that we have updates to apply
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No updates provided", []global.ValidationError{
			{Field: "body", Message: "Request body must contain at least one field to update", Code: "empty_updates"},
		}))
		return
	}

	// Prevent updating immutable fields - remove them from updates instead of erroring
	immutableFields := []string{"_id", "id", "order_number", "created_at", "customer_id", "customer_email"}
	for _, field := range immutableFields {
		if _, exists := updates[field]; exists {
			delete(updates, field)
			log.Printf("Warning: Removed immutable field '%s' from update request", field)
		}
	}

	// Check if we still have updates after removing immutable fields
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("No valid updates provided", []global.ValidationError{
			{Field: "body", Message: "All provided fields are immutable and cannot be updated", Code: "no_valid_updates"},
		}))
		return
	}

	// Update the order in MongoDB
	updatedOrder, err := mongo.UpdateOrderByNumber(ctx, orderNumber, updates)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "mongo: no documents in result" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Order not found", []global.ValidationError{
				{Field: "order_number", Message: "No order exists with this order number", Code: "not_found"},
			}))
			return
		}
		// Other database error
		log.Printf("Error updating order in MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to update order", nil))
		return
	}

	// Return the updated order
	c.JSON(http.StatusOK, global.SuccessResponse(updatedOrder))
}

// DeleteOrderByNumber deletes an order by order number from the database
func DeleteOrderByNumber(c *gin.Context) {
	orderNumber := c.Param("orderNumber")

	// Validate order number format
	if len(orderNumber) < 3 || len(orderNumber) > 100 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid order number format", []global.ValidationError{
			{Field: "order_number", Message: "Order number must be between 3 and 100 characters", Code: "invalid_format"},
		}))
		return
	}

	ctx := c.Request.Context()

	// Delete the order from MongoDB (this also returns the deleted order for response)
	deletedOrder, err := mongo.DeleteOrderByNumber(ctx, orderNumber)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "mongo: no documents in result" || err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Order not found", []global.ValidationError{
				{Field: "order_number", Message: "No order exists with this order number", Code: "not_found"},
			}))
			return
		}
		// Other database error
		log.Printf("Error deleting order from MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to delete order", nil))
		return
	}

	// Return success with the deleted order info
	c.JSON(http.StatusOK, global.SuccessResponse(map[string]interface{}{
		"deleted_order_number": deletedOrder.OrderNumber,
		"message":              "Order successfully deleted",
	}))
}

// GetAllCategories retrieves all distinct categories from products
func GetAllCategories(c *gin.Context) {
	categories, err := mongo.GetAllCategories()
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to fetch categories", nil))
		return
	}

	// Return categories with count information
	response := map[string]interface{}{
		"categories":  categories,
		"total_count": len(categories),
	}

	c.JSON(http.StatusOK, global.SuccessResponse(response))
}

func GetAllCustomers(c *gin.Context) {
	customers, err := mongo.GetAllCustomers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to retrieve customers: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(customers))
}

func GetAllReviews(c *gin.Context) {}

func GetAllCartItems(c *gin.Context) {}

func GetBaseAnalytics(c *gin.Context) {}

func GetInventoryPagenated(c *gin.Context) {}

func GetCustomerSegments(c *gin.Context) {
	segments, err := mongo.GetCustomerSpendingSegments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to fetch customer segments", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(segments))
}

func GetCustomerOrders(c *gin.Context) {
	customerID := c.Param("id")

	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid customer ID format", []global.ValidationError{
			{Field: "id", Message: "Must be a valid MongoDB ObjectID", Code: "invalid_format"},
		}))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	result, err := mongo.GetCustomerOrdersWithStats(objectID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to fetch customer orders", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(result))
}

func CreateCustomer(c *gin.Context) {
	var req models.CreateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid request data", []global.ValidationError{
			{Field: "request", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to process password", nil))
		return
	}

	customer := &models.Customer{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Addresses: []models.Address{req.Address},
		Preferences: models.Preferences{
			Newsletter:         true,
			SMSNotifications:   false,
			EmailNotifications: true,
			Language:           "en",
			Currency:           "CAD",
			FavoriteCategories: []string{},
		},
		LoyaltyPoints: 0,
		AccountStatus: "active",
		EmailVerified: false,
		PhoneVerified: false,
		TotalOrders:   0,
		TotalSpent:    0.0,
	}
	customer.SetTimestamps()

	customer.Addresses[0].IsDefault = true

	createdCustomer, err := mongo.CreateCustomer(c.Request.Context(), customer)
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, global.ErrorResponse("Email already registered", []global.ValidationError{
				{Field: "email", Message: "This email is already in use", Code: "duplicate_email"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to create customer", nil))
		return
	}

	// simulate: Send welcome email (optional)

	c.JSON(http.StatusCreated, global.SuccessResponse(createdCustomer))
}

func GetCustomerByID(c *gin.Context) {
	customerID := c.Param("id")

	// Validate ObjectID format
	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid customer ID format", []global.ValidationError{
			{Field: "id", Message: "Must be a valid MongoDB ObjectID", Code: "invalid_format"},
		}))
		return
	}

	// In Production, this would be protected to allow only the customer themselves or admins to access the data

	// Fetch customer from database
	customer, err := mongo.GetCustomerByID(c.Request.Context(), objectID)
	if err != nil {
		if err.Error() == "customer not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Customer not found", []global.ValidationError{
				{Field: "id", Message: "No customer exists with this ID", Code: "not_found"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to fetch customer", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(customer))
}

func UpdateCustomer(c *gin.Context) {
	customerID := c.Param("id")

	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid customer ID format", []global.ValidationError{
			{Field: "id", Message: "Must be a valid MongoDB ObjectID", Code: "invalid_format"},
		}))
		return
	}

	// In Production, this would be protected to allow only the customer themselves or admins to access the data

	// Bind request payload
	var req models.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid request data", []global.ValidationError{
			{Field: "request", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	updatedCustomer, err := mongo.UpdateCustomer(c.Request.Context(), objectID, &req)
	if err != nil {
		if err.Error() == "customer not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Customer not found", []global.ValidationError{
				{Field: "id", Message: "No customer exists with this ID", Code: "not_found"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to update customer", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(updatedCustomer))
}

func AddCustomerAddress(c *gin.Context) {
	customerID := c.Param("id")

	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid customer ID format", []global.ValidationError{
			{Field: "id", Message: "Must be a valid MongoDB ObjectID", Code: "invalid_format"},
		}))
		return
	}

	// In Production, this would be protected to allow only the customer themselves or admins to access the data
	var address models.Address
	if err := c.ShouldBindJSON(&address); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid address data", []global.ValidationError{
			{Field: "address", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	updatedCustomer, err := mongo.AddCustomerAddress(c.Request.Context(), objectID, address)
	if err != nil {
		if err.Error() == "customer not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Customer not found", []global.ValidationError{
				{Field: "id", Message: "No customer exists with this ID", Code: "not_found"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to add address", nil))
		return
	}

	c.JSON(http.StatusCreated, global.SuccessResponse(updatedCustomer))
}

func UpdateCustomerAddress(c *gin.Context) {
	customerID := c.Param("id")
	addressIndex, err := strconv.Atoi(c.Param("addressId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid address ID", []global.ValidationError{
			{Field: "addressId", Message: "Must be a valid integer index", Code: "invalid_format"},
		}))
		return
	}

	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid customer ID format", []global.ValidationError{
			{Field: "id", Message: "Must be a valid MongoDB ObjectID", Code: "invalid_format"},
		}))
		return
	}

	// In Production, this would be protected to allow only the customer themselves or admins to access the data
	var address models.Address
	if err := c.ShouldBindJSON(&address); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid address data", []global.ValidationError{
			{Field: "address", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	updatedCustomer, err := mongo.UpdateCustomerAddress(c.Request.Context(), objectID, addressIndex, address)
	if err != nil {
		if err.Error() == "customer not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Customer not found", []global.ValidationError{
				{Field: "id", Message: "No customer exists with this ID", Code: "not_found"},
			}))
			return
		}
		if err.Error() == "address not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Address not found", []global.ValidationError{
				{Field: "addressId", Message: "No address exists at this index", Code: "not_found"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to update address", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(updatedCustomer))
}

func DeleteCustomerAddress(c *gin.Context) {
	customerID := c.Param("id")
	addressIndex, err := strconv.Atoi(c.Param("addressId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid address ID", []global.ValidationError{
			{Field: "addressId", Message: "Must be a valid integer index", Code: "invalid_format"},
		}))
		return
	}

	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid customer ID format", []global.ValidationError{
			{Field: "id", Message: "Must be a valid MongoDB ObjectID", Code: "invalid_format"},
		}))
		return
	}

	// TODO: Authorization - verify user owns this customer profile

	updatedCustomer, err := mongo.DeleteCustomerAddress(c.Request.Context(), objectID, addressIndex)
	if err != nil {
		if err.Error() == "customer not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Customer not found", []global.ValidationError{
				{Field: "id", Message: "No customer exists with this ID", Code: "not_found"},
			}))
			return
		}
		if err.Error() == "address not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Address not found", []global.ValidationError{
				{Field: "addressId", Message: "No address exists at this index", Code: "not_found"},
			}))
			return
		}
		if err.Error() == "cannot delete last address" {
			c.JSON(http.StatusBadRequest, global.ErrorResponse("Cannot delete last address", []global.ValidationError{
				{Field: "addressId", Message: "Customer must have at least one address", Code: "invalid_operation"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to delete address", nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(updatedCustomer))
}

// DeleteCustomer removes a customer by ID
func DeleteCustomer(c *gin.Context) {
	customerID := c.Param("id")

	// Validate customer ID format by trying to parse it
	_, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid customer ID format", []global.ValidationError{
			{Field: "id", Message: "id must be a valid ObjectID"},
		}))
		return
	}

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Delete customer from database
	err = mongo.DeleteCustomer(ctx, customerID)
	if err != nil {
		if err.Error() == "customer not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Customer not found", []global.ValidationError{
				{Field: "id", Message: "customer with this ID does not exist"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to delete customer: "+err.Error(), nil))
		return
	}

	// Return minimal response (just ID) following the response optimization pattern
	c.JSON(http.StatusOK, global.SuccessResponse(map[string]string{
		"id": customerID,
	}))
}

func GetReviewsForItem(c *gin.Context) {
	// Get entity type and ID from context (set by ReviewsMiddleware)
	entityType, exists := c.Get("entity")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity type not found in context", nil))
		return
	}

	entityID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity ID not found in context", nil))
		return
	}

	// Convert to strings
	entityTypeStr, ok := entityType.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity type format", nil))
		return
	}

	entityIDStr, ok := entityID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity ID format", nil))
		return
	}

	// Get reviews from database
	reviews, err := mongo.GetAllReviewsForItem(entityTypeStr, entityIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to retrieve reviews: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(reviews))
}

func CreateReviewForItem(c *gin.Context) {
	// Get entity type and ID from context (set by ReviewsMiddleware)
	entityType, exists := c.Get("entity")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity type not found in context", nil))
		return
	}

	entityID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity ID not found in context", nil))
		return
	}

	// Convert to strings
	entityTypeStr, ok := entityType.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity type format", nil))
		return
	}

	entityIDStr, ok := entityID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity ID format", nil))
		return
	}

	// Only allow creating reviews for products
	if entityTypeStr != "product" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Reviews can only be created for products", []global.ValidationError{
			{Field: "entity", Message: "entity type must be 'product' for review creation"},
		}))
		return
	}

	// Parse request body
	var reviewRequest models.CreateReviewRequest
	if err := c.ShouldBindJSON(&reviewRequest); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid request data", []global.ValidationError{
			{Field: "request", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	// Set the product ID from the entity ID in URL
	productObjID, err := bson.ObjectIDFromHex(entityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid product ID format", []global.ValidationError{
			{Field: "id", Message: "product ID must be a valid ObjectID hex string"},
		}))
		return
	}
	reviewRequest.ProductID = productObjID

	// Create review in database
	review, err := mongo.CreateReviewForItem(&reviewRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to create review: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusCreated, global.SuccessResponse(review))
}
func UpdateReviewForItem(c *gin.Context) {
	// Get entity type and ID from context (set by ReviewsMiddleware)
	entityType, exists := c.Get("entity")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity type not found in context", nil))
		return
	}

	entityID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity ID not found in context", nil))
		return
	}

	// Get review ID from query parameter
	reviewID := c.Query("reviewId")
	if reviewID == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Review ID is required", []global.ValidationError{
			{Field: "reviewId", Message: "reviewId query parameter is required"},
		}))
		return
	}

	// Convert to strings
	entityTypeStr, ok := entityType.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity type format", nil))
		return
	}

	entityIDStr, ok := entityID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity ID format", nil))
		return
	}

	// Only allow updating reviews for products
	if entityTypeStr != "product" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Reviews can only be updated for products", []global.ValidationError{
			{Field: "entity", Message: "entity type must be 'product' for review updates"},
		}))
		return
	}

	// Parse request body
	var updateRequest models.UpdateReviewRequest
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid request data", []global.ValidationError{
			{Field: "request", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	// Update review in database
	updatedReview, err := mongo.UpdateReviewForItem(reviewID, entityIDStr, &updateRequest)
	if err != nil {
		if err.Error() == "review not found" || err.Error() == "review not found for this product" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Review not found", []global.ValidationError{
				{Field: "reviewId", Message: "review not found or does not belong to this product"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to update review: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(updatedReview))
}
func DeleteReviewForItem(c *gin.Context) {
	// Get entity type and ID from context (set by ReviewsMiddleware)
	entityType, exists := c.Get("entity")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity type not found in context", nil))
		return
	}

	entityID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Entity ID not found in context", nil))
		return
	}

	// Get review ID from query parameter
	reviewID := c.Query("reviewId")
	if reviewID == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Review ID is required", []global.ValidationError{
			{Field: "reviewId", Message: "reviewId query parameter is required"},
		}))
		return
	}

	// Convert to strings
	entityTypeStr, ok := entityType.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity type format", nil))
		return
	}

	entityIDStr, ok := entityID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Invalid entity ID format", nil))
		return
	}

	// Only allow deleting reviews for products
	if entityTypeStr != "product" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Reviews can only be deleted for products", []global.ValidationError{
			{Field: "entity", Message: "entity type must be 'product' for review deletion"},
		}))
		return
	}

	// Delete review from database
	deletedReviewID, err := mongo.DeleteReviewForItem(reviewID, entityIDStr)
	if err != nil {
		if err.Error() == "review not found" || err.Error() == "review not found for this product" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Review not found", []global.ValidationError{
				{Field: "reviewId", Message: "review not found or does not belong to this product"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to delete review: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted_review_id": deletedReviewID,
		"message":           "Review successfully deleted",
	})
}

// SearchDatabase searches across all collections and groups results by type
func SearchDatabase(c *gin.Context) {
	// Get search query parameter
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Search query is required", []global.ValidationError{
			{Field: "q", Message: "q query parameter is required"},
		}))
		return
	}

	// Get optional limit parameter (default: 10 per collection)
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// Perform search across all collections
	results, err := mongo.SearchDatabase(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Search failed: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"query":    query,
		"limit":    limit,
		"results":  results,
		"searched": []string{"products", "customers", "orders", "reviews"},
	})
}

// GetSalesAnalytics returns daily sales summary with optional date range filtering
func GetSalesAnalytics(c *gin.Context) {
	// Get optional date range parameters
	startDateStr := c.Query("start_date")           // Format: 2025-11-01
	endDateStr := c.Query("end_date")               // Format: 2025-11-30
	groupByStr := c.DefaultQuery("group_by", "day") // day, week, month

	// Validate group_by parameter
	if groupByStr != "day" && groupByStr != "week" && groupByStr != "month" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid group_by parameter", []global.ValidationError{
			{Field: "group_by", Message: "group_by must be one of: day, week, month"},
		}))
		return
	}

	// Get sales analytics from database
	salesData, err := mongo.GetSalesAnalytics(startDateStr, endDateStr, groupByStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to retrieve sales analytics: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"group_by":   groupByStr,
		"start_date": startDateStr,
		"end_date":   endDateStr,
		"data":       salesData,
	})
}

// GetTopProducts returns top N products by revenue or quantity
func GetTopProducts(c *gin.Context) {
	// Get query parameters
	limitStr := c.DefaultQuery("limit", "10")
	sortBy := c.DefaultQuery("sortBy", "revenue")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	// Parse limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid limit parameter", []global.ValidationError{
			{Field: "limit", Message: "limit must be a number between 1 and 100"},
		}))
		return
	}

	// Validate sortBy parameter
	validSortBy := map[string]bool{
		"revenue":  true,
		"quantity": true,
	}
	if !validSortBy[sortBy] {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid sortBy parameter", []global.ValidationError{
			{Field: "sortBy", Message: "sortBy must be either 'revenue' or 'quantity'"},
		}))
		return
	}

	// Get top products data
	topProducts, err := mongo.GetTopProductsByRevenue(limit, sortBy, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to retrieve top products: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(topProducts))
}

// GetInventoryAnalytics returns real-time inventory status with optional alerts filter
func GetInventoryAnalytics(c *gin.Context) {
	// Get query parameters
	alertsOnlyStr := c.DefaultQuery("alertsOnly", "false")

	// Parse alertsOnly parameter
	alertsOnly := false
	if alertsOnlyStr == "true" || alertsOnlyStr == "1" {
		alertsOnly = true
	}

	// Get inventory status data
	inventoryStatus, err := mongo.GetInventoryStatus(alertsOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to retrieve inventory status: "+err.Error(), nil))
		return
	}

	// Add summary metadata
	response := map[string]interface{}{
		"inventory": inventoryStatus,
		"summary": map[string]interface{}{
			"total_products": len(inventoryStatus),
			"alerts_only":    alertsOnly,
		},
	}

	c.JSON(http.StatusOK, global.SuccessResponse(response))
}

// Cart handlers

// GetCart retrieves cart by session ID
func GetCart(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Session ID is required", []global.ValidationError{
			{Field: "sessionId", Message: "sessionId URL parameter is required"},
		}))
		return
	}

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	cart, err := redis.GetCart(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to retrieve cart: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(cart))
}

// AddToCart adds an item to the cart
func AddToCart(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Session ID is required", []global.ValidationError{
			{Field: "sessionId", Message: "sessionId URL parameter is required"},
		}))
		return
	}

	var request models.AddToCartRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid request data", []global.ValidationError{
			{Field: "request", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Get product details by SKU
	product, err := mongo.GetProductBySKU(ctx, request.SKU)
	if err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Product not found", []global.ValidationError{
				{Field: "sku", Message: "product with this SKU does not exist"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to retrieve product: "+err.Error(), nil))
		return
	}

	// Check stock availability
	if product.Stock.Total < request.Quantity {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Insufficient stock", []global.ValidationError{
			{Field: "quantity", Message: fmt.Sprintf("only %d items available in stock", product.Stock.Total)},
		}))
		return
	}

	// Add to cart
	cart, err := redis.AddToCart(ctx, sessionID, request.SKU, request.Quantity, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to add item to cart: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusCreated, global.SuccessResponse(cart))
}

// UpdateCartItem updates the quantity of an item in the cart
func UpdateCartItem(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Session ID is required", []global.ValidationError{
			{Field: "sessionId", Message: "sessionId URL parameter is required"},
		}))
		return
	}

	sku := c.Param("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("SKU is required", []global.ValidationError{
			{Field: "sku", Message: "sku parameter is required"},
		}))
		return
	}

	var request models.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Invalid request data", []global.ValidationError{
			{Field: "request", Message: err.Error(), Code: "validation_error"},
		}))
		return
	}

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Update cart item
	cart, err := redis.UpdateCartItem(ctx, sessionID, sku, request.Quantity)
	if err != nil {
		if err.Error() == "item not found in cart" {
			c.JSON(http.StatusNotFound, global.ErrorResponse("Item not found in cart", []global.ValidationError{
				{Field: "sku", Message: "item with this SKU does not exist in cart"},
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to update cart item: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(cart))
}

// RemoveFromCart removes an item from the cart
func RemoveFromCart(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Session ID is required", []global.ValidationError{
			{Field: "sessionId", Message: "sessionId URL parameter is required"},
		}))
		return
	}

	sku := c.Param("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("SKU is required", []global.ValidationError{
			{Field: "sku", Message: "sku parameter is required"},
		}))
		return
	}

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Remove from cart
	cart, err := redis.RemoveFromCart(ctx, sessionID, sku)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to remove item from cart: "+err.Error(), nil))
		return
	}

	// Return minimal response following optimization pattern
	c.JSON(http.StatusOK, global.SuccessResponse(map[string]interface{}{
		"sku":     sku,
		"removed": true,
		"cart":    cart,
	}))
}

// ClearCart removes all items from the cart
func ClearCart(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, global.ErrorResponse("Session ID is required", []global.ValidationError{
			{Field: "sessionId", Message: "sessionId URL parameter is required"},
		}))
		return
	}

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	err := redis.ClearCart(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to clear cart: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, global.SuccessResponse(map[string]interface{}{
		"session_id": sessionID,
		"cleared":    true,
	}))
}

// AI Analytics Handlers

// GenerateAISalesReport generates AI-powered sales analytics report
func GenerateAISalesReport(c *gin.Context) {
	// Get date range parameters
	startDate := c.DefaultQuery("startDate", "")
	endDate := c.DefaultQuery("endDate", "")

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Generate AI sales report
	report, err := ai.GenerateSalesReport(ctx, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to generate sales report: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, report)
}

// GenerateAICustomerInsights generates AI-powered customer analytics
func GenerateAICustomerInsights(c *gin.Context) {
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Generate AI customer insights
	report, err := ai.GenerateCustomerInsights(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to generate customer insights: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, report)
}

// GenerateAIInventoryReport generates AI-powered inventory analytics
func GenerateAIInventoryReport(c *gin.Context) {
	// Get alerts filter parameter
	alertsOnlyStr := c.DefaultQuery("alertsOnly", "false")
	alertsOnly := alertsOnlyStr == "true" || alertsOnlyStr == "1"

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Generate AI inventory report
	report, err := ai.GenerateInventoryReport(ctx, alertsOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to generate inventory report: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, report)
}

// GenerateAIProductAnalysis generates AI-powered top products analysis
func GenerateAIProductAnalysis(c *gin.Context) {
	// Get query parameters
	limitStr := c.DefaultQuery("limit", "10")
	sortBy := c.DefaultQuery("sortBy", "revenue")
	startDate := c.DefaultQuery("startDate", "")
	endDate := c.DefaultQuery("endDate", "")

	// Parse limit parameter
	limit := 10
	if limitValue, err := strconv.Atoi(limitStr); err == nil && limitValue > 0 && limitValue <= 100 {
		limit = limitValue
	}

	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Generate AI product analysis
	report, err := ai.GenerateTopProductsAnalysis(ctx, limit, sortBy, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, global.ErrorResponse("Failed to generate product analysis: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, report)
}
