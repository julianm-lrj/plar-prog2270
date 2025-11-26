package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
