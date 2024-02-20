package main

import (
	"api_runner/handlers"
	"fmt"
	"github.com/gin-gonic/gin"
	"os/exec"
	"runtime"
)

func main() {

	// open port 80
	if runtime.GOOS != "windows" {
		cmd := exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", "80", "-j", "ACCEPT")
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
	}
	handlers.InitMongo()
	router := gin.Default()
	router.GET("/vm", handlers.GetAllContainersFromVM)
	router.GET("/vm/:id/containers", handlers.GetContainersFromVM)
	router.GET("/ping", handlers.Ping)
	router.POST("/vm", handlers.CreateContainer)
	router.POST("/build", handlers.BuildImage)
	router.DELETE("/vm/:id", handlers.DeleteContainer)

	fmt.Println("Server is running on localhost:80")
	router.Run(":80")
}
