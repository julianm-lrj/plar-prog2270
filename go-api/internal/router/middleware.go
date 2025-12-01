package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
)

func ReviewsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		entityType := c.Request.URL.Query().Get("item")
		if entityType == "" {
			c.JSON(http.StatusBadRequest, global.ErrorResponse("item query parameter required", []global.ValidationError{
				{Field: "item", Message: "item query parameter is required", Code: "required"},
			}))
			c.Abort()
			return
		}

		entityId := c.Request.URL.Query().Get("id")
		if entityId == "" {
			c.JSON(http.StatusBadRequest, global.ErrorResponse("id query parameter required", []global.ValidationError{
				{Field: "id", Message: "id query parameter is required", Code: "required"},
			}))
			c.Abort()
			return
		}

		c.Set("entity", entityType)
		c.Set("id", entityId)
		c.Next()
	}
}
