package main

import (
	"net/http"
	"net/url"
	"strings"

	"pack_mate/internal/config"
	"pack_mate/internal/handler"
	mongodb "pack_mate/internal/repository/mongodb"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func newRouter(cfg *config.Configuration, client *cos.Client, bucketURL *url.URL, db *mongo.Database) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware(cfg.CORS.AllowedOrigins))

	itemRepo := mongodb.NewItemRepository(db)
	uploadService := service.NewCOSUploadService(client, bucketURL)
	itemService := service.NewItemService(itemRepo, uploadService)

	registerAPIRoutes(router, itemService)

	return router
}

func registerAPIRoutes(router gin.IRoutes, itemService service.ItemService) {
	itemHandler := handler.NewItemHandler(itemService)

	router.POST("/api/v1/item", itemHandler.CreateItem)
	router.GET("/api/v1/item", itemHandler.ListItems)
	router.GET("/api/v1/item/:item_id", itemHandler.GetItem)
	router.PUT("/api/v1/item/:item_id", itemHandler.UpdateItem)
	router.DELETE("/api/v1/item/:item_id", itemHandler.DeleteItem)
}

func corsMiddleware(origins []string) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		allowedOrigins[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
