package main

import (
	"api_runner/handlers"
	"fmt"
	"github.com/gin-gonic/gin"
	"os/exec"
)

func main() {

	// open port 80
	cmd := exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", "80", "-j", "ACCEPT")
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	router := gin.Default()
	router.GET("/vm", handlers.GetAllContainersFromVM)
	router.GET("/vm/:id/containers", handlers.GetContainersFromVM)
	router.POST("/vm", handlers.CreateContainer)
	router.POST("/build", handlers.BuildImage)
	router.DELETE("/vm/:id", handlers.DeleteContainer)

	fmt.Println("Server is running on localhost:80")
	router.Run("localhost:80")
}
