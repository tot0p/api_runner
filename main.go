package main

import (
	"api_runner/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/vm", handlers.GetVMs)
	router.GET("/vm/:id/containers", handlers.GetContainers)
	router.POST("/vm", handlers.CreateVM)
	router.DELETE("/vm/:id", handlers.DeleteInstance)

	router.Run("localhost:8080")
}
