package router

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
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
