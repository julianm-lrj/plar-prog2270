package router

import (
	"github.com/gin-gonic/gin"
)

var Router *gin.Engine

func InitEngine() {
	Router = gin.Default()
}

func InitializeRoutes() {
	api := Router.Group("/api")
	{
		api.GET("/health", nil)

		products := api.Group("/products")
		{
			products.GET("/", nil)
			products.POST("/", nil)
			products.GET("/:id", nil)
			products.PUT("/:id", nil)
			products.DELETE("/:id", nil)
			products.GET("/categories", nil)
			products.POST("/categories", nil)
			products.GET("/categories/:id", nil)
			products.PUT("/categories/:id", nil)
			products.DELETE("/categories/:id", nil)
			products.GET("/search", nil)
		}

		orders := api.Group("/orders")
		{
			orders.GET("/", nil)
			orders.POST("/", nil)
			orders.GET("/:id", nil)
			orders.PUT("/:id", nil)
			orders.DELETE("/:id", nil)
		}

		customers := api.Group("/customers")
		{
			customers.GET("/", nil)
			customers.POST("/", nil)
			customers.GET("/:id", nil)
			customers.PUT("/:id", nil)
			customers.DELETE("/:id", nil)
		}

		reviews := api.Group("/reviews")
		{
			reviews.GET("/", nil)
			reviews.POST("/", nil)
			reviews.GET("/item/:id", nil)
			reviews.GET("/customer/:id", nil)
			reviews.PUT("/:id", nil)
			reviews.DELETE("/item/:id", nil)
			reviews.DELETE("/customer/:id", nil)
		}

		cart := api.Group("/cart")
		{
			cart.GET("/", nil)
			cart.POST("/", nil)
			cart.PUT("/:id", nil)
			cart.DELETE("/:id", nil)
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
			analytics.GET("/", nil)
		}

		admin := api.Group("/admin")
		{
			admin.GET("/", nil)
		}
	}
}
