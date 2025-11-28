package router

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
	"julianmorley.ca/con-plar/prog2270/pkg/models"
	"julianmorley.ca/con-plar/prog2270/pkg/mongo"
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
		"deleted_product": deletedProduct,
		"message":         "Product successfully deleted",
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

func GetAllOrders(c *gin.Context) {}

func GetAllCustomers(c *gin.Context) {}

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
