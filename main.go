package main

import (
	"api_runner/handlers"
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/vm", handlers.GetAllContainersFromVM)
	router.GET("/vm/:id/containers", handlers.GetContainersFromVM)
	router.POST("/vm", handlers.CreateContainer)
	router.POST("/build", handlers.BuildImage)
	router.DELETE("/vm/:id", handlers.DeleteContainer)

	fmt.Println("Server is running on localhost:8080")
	router.Run("localhost:8080")
}
