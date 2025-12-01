package router

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var Router *gin.Engine

func InitEngine() {
	Router = gin.Default()
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "https://plar-conestoga-prog2270.julianmorley.ca"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}

func InitializeRoutes() {
	api := Router.Group("/api")
	{
		api.GET("/health", HealthCheck)
		api.GET("/search", SearchDatabase)

		products := api.Group("/products")
		{
			products.GET("/", GetAllProducts)
			products.POST("/", CreateNewProducts)
			products.PUT("/", BulkEditProducts)
			products.DELETE("/", BulkDeleteProducts)
			products.GET("/:sku", GetProductBySKU)
			products.PUT("/:sku", EditProductBySKU)
			products.DELETE("/:sku", DeleteProductBySKU)
		}

		categories := api.Group("/categories")
		{
			categories.GET("/", GetAllCategories)
		}

		orders := api.Group("/orders")
		{
			orders.GET("/", GetAllOrders)
			orders.POST("/", CreateNewOrders)
			orders.PUT("/", BulkEditOrders)
			orders.DELETE("/", BulkDeleteOrders)
			orders.GET("/:orderNumber", GetOrderByNumber)
			orders.PUT("/:orderNumber", EditOrderByNumber)
			orders.DELETE("/:orderNumber", DeleteOrderByNumber)
		}

		customers := api.Group("/customers")
		{
			customers.GET("/", GetAllCustomers)
			customers.POST("/", CreateCustomer)
			customers.GET("/:id", GetCustomerByID)
			customers.PUT("/:id", UpdateCustomer)
			customers.DELETE("/:id", DeleteCustomer)
			customers.GET("/:id/orders", GetCustomerOrders)
			customers.POST("/:id/addresses", AddCustomerAddress)
			customers.PUT("/:id/addresses/:addressId", UpdateCustomerAddress)
			customers.DELETE("/:id/addresses/:addressId", DeleteCustomerAddress)
		}

		reviews := api.Group("/reviews")
		reviews.Use(ReviewsMiddleware())
		{
			reviews.GET("/", GetReviewsForItem)
			reviews.POST("/", CreateReviewForItem)
			reviews.PUT("/", UpdateReviewForItem)
			reviews.DELETE("/", DeleteReviewForItem)
		}

		cart := api.Group("/cart")
		{
			cart.GET("/:sessionId", GetCart)
			cart.POST("/:sessionId/items", AddToCart)
			cart.PUT("/:sessionId/items/:sku", UpdateCartItem)
			cart.DELETE("/:sessionId/items/:sku", RemoveFromCart)
			cart.DELETE("/:sessionId/clear", ClearCart)
		}

		inventory := api.Group("/inventory")
		{
			inventory.GET("/", nil)
			inventory.POST("/", nil)
			inventory.GET("/:id", nil)
			inventory.PUT("/:id", nil)
		}

		analytics := api.Group("/analytics")
		{
			analytics.GET("/sales", GetSalesAnalytics)
			analytics.GET("/customers/segments", GetCustomerSegments)
			analytics.GET("/top-products", GetTopProducts)
			analytics.GET("/inventory", GetInventoryAnalytics)

			// AI-powered analytics endpoints
			aiAnalytics := analytics.Group("/ai")
			{
				aiAnalytics.GET("/sales-report", GenerateAISalesReport)
				aiAnalytics.GET("/customer-insights", GenerateAICustomerInsights)
				aiAnalytics.GET("/inventory-report", GenerateAIInventoryReport)
				aiAnalytics.GET("/product-analysis", GenerateAIProductAnalysis)
			}
		}

		admin := api.Group("/admin")
		{
			admin.GET("/", nil)
		}
	}
}
