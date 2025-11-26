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

func GetAllOrders(c *gin.Context) {}

func GetAllCustomers(c *gin.Context) {}

func GetAllReviews(c *gin.Context) {}

func GetAllCartItems(c *gin.Context) {}

func GetBaseAnalytics(c *gin.Context) {}

func GetInventoryPagenated(c *gin.Context) {}

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

	// Validate ObjectID format
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
