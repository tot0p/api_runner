package handlers

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"io"
	"net"
	"net/http"
	"strconv"
)

type VM struct {
	ID     string `json:"id"`
	Region string `json:"region"`
}

var VMs = []VM{
	{ID: "1", Region: "usa"},
	{ID: "2", Region: "fra"},
	{ID: "3", Region: "ger"},
	{ID: "4", Region: "usa"},
}

type Container struct {
	ID     string `json:"Id"`
	IP     string `json:"IPAddress"`
	Name   string `json:"Names"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Image  string `json:"Image"`
}

func GetVMs(c *gin.Context) {
	containers := GetContainerFromVM()
	var containerList []Container
	for _, element := range containers {
		containerList = append(containerList, Container{
			ID:     element.ID,
			IP:     element.NetworkSettings.Networks["bridge"].IPAddress,
			Name:   element.Names[0],
			State:  element.State,
			Status: element.Status,
			Image:  element.Image,
		})
	}
	c.IndentedJSON(http.StatusOK, containerList)
}

func GetContainers(c *gin.Context) {
	id := c.Param("id")
	for _, vm := range VMs {
		if vm.ID == id {
			c.IndentedJSON(http.StatusOK, vm)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "container not found"})
}

func CreateVM(c *gin.Context) {
	var newVM VM
	if err := c.ShouldBindJSON(&newVM); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	VMs = append(VMs, newVM)
	c.IndentedJSON(http.StatusCreated, newVM)
}

func DeleteInstance(c *gin.Context) {
	id := c.Param("id")
	for i, instance := range VMs {
		if instance.ID == id {
			VMs = append(VMs[:i], VMs[i+1:]...)
			c.IndentedJSON(http.StatusOK, gin.H{"message": "instance deleted"})
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "instance not found"})
}

func GetContainerFromVM() []types.Container {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		panic(err)
	}
	return containers
}

func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func InstanciateContainer(imageName, containerName string, portBindings []int) (map[int]int, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())
	portBinding := make(nat.PortMap)
	for i := 0; i < len(portBindings); i++ {
		port := portBindings[i]
		freePort, err := GetFreePort()
		if err != nil {
			return nil, err
		}
		hostBinding := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: fmt.Sprintf("%d", freePort),
		}
		containerPort, err := nat.NewPort("tcp", fmt.Sprintf("%d", port))
		if err != nil {
			panic("Unable to get the port")
		}
		portBinding[containerPort] = []nat.PortBinding{hostBinding}
	}
	fmt.Println(portBinding)

	// check if the image exists
	_, _, err = cli.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil {
		// pull the image if it doesn't exist
		reader, err := cli.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		_, err = io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		_, _, err = cli.ImageInspectWithRaw(context.Background(), imageName)
		if err != nil {
			return nil, err
		}
	}

	// create a container
	cont, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: imageName,
		},
		&container.HostConfig{
			PortBindings: portBinding,
		},
		nil,
		nil,
		containerName,
	)
	if err != nil {
		return nil, err
	}
	fmt.Println(cont.ID)
	// start the container
	if err := cli.ContainerStart(context.Background(), cont.ID, container.StartOptions{}); err != nil {
		return nil, err
	}
	// inspect the container
	inspect, err := cli.ContainerInspect(context.Background(), cont.ID)
	if err != nil {
		return nil, err
	}
	fmt.Println(inspect.NetworkSettings.Ports)
	ext := make(map[int]int)
	for k, v := range portBinding {
		port, err := strconv.Atoi(v[0].HostPort)
		if err != nil {
			return nil, err
		}
		ext[int(k.Int())] = port
	}
	return ext, nil
}
