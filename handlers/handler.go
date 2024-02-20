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
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type Container struct {
	ID     string `json:"Id"`
	IP     string `json:"IPAddress"`
	Name   string `json:"Names"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Image  string `json:"Image"`
}

type ContainerRequest struct {
	Link string `json:"link"`
}

type ImageRequest struct {
	RepositoryURL string `json:"repository"`
}

func GetAllContainersFromVM(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to create docker client"})
		return
	}
	cli.NegotiateAPIVersion(context.Background())
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to list the containers"})
		return
	}
	var containerList []Container
	for _, contained := range containers {
		containerList = append(containerList, Container{
			ID:     contained.ID,
			IP:     contained.NetworkSettings.Networks["bridge"].IPAddress,
			Name:   contained.Names[0],
			State:  contained.State,
			Status: contained.Status,
			Image:  contained.Image,
		})
	}
	c.IndentedJSON(http.StatusOK, containerList)
}

func GetContainersFromVM(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to create docker client"})
		return
	}
	cli.NegotiateAPIVersion(context.Background())
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to list the containers"})
		return
	}

	id := c.Param("id")
	for _, contained := range containers {
		if contained.ID == id {
			c.IndentedJSON(http.StatusOK, contained)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "container not found"})
}

// CreateContainer creates a container and starts it
func CreateContainer(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to create docker client"})
		return
	}
	cli.NegotiateAPIVersion(context.Background())

	// To set a specific image version, use the following format: imageName:version
	var containerRequest ContainerRequest
	err = c.BindJSON(&containerRequest)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	pattern := `https:\/\/github\.com\/\w+\/(\w+\.git)`

	// Compile the regular expression pattern
	re := regexp.MustCompile(pattern)
	var imageName string
	if re.MatchString(containerRequest.Link) {
		// Find the matched string
		match := re.FindStringSubmatch(containerRequest.Link)
		imageName = match[1]
	} else {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid repository link"})
		return
	}

	// check if the image exists
	_, _, err = cli.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil {
		// build the image from the repository link
		cmd := exec.Command("docker", "build", containerRequest.Link, "-t", imageName)
		err = cmd.Run()
		if err != nil {
			fmt.Println("Unable to build the image from: ", containerRequest.Link)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to build the image"})
			return
		}
		// check if the image exists
		_, _, err = cli.ImageInspectWithRaw(context.Background(), imageName)
		if err != nil {
			fmt.Println("Unable to find the image: ", imageName)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to find the image"})
			return
		}
	}

	// list the ports of the image
	inspect, _, err := cli.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil {
		fmt.Println("Unable to inspect the image")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to inspect the image"})
		return
	}

	exposedPorts := make(nat.PortMap)
	for k := range inspect.Config.ExposedPorts {
		// get a free port
		port, err := FindFreePort()
		if err != nil {
			fmt.Println("Unable to find a free port")
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to find a free port"})
			return
		}
		hostBinding := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: fmt.Sprintf("%d", port),
		}
		containerPort, err := nat.NewPort("tcp", fmt.Sprintf("%d", k.Int()))
		if err != nil {
			fmt.Println("Unable to get the port")
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to get the port"})
			return
		}
		if runtime.GOOS != "windows" {
			cmd := exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", fmt.Sprintf("%d", port), "-j", "ACCEPT")
			err = cmd.Run()
			if err != nil {
				panic(err)
			}
		}

		exposedPorts[containerPort] = []nat.PortBinding{hostBinding}
		fmt.Println("port: ", k.Int(), "was mapped to: ", port)
	}

	// Create a container
	cont, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: imageName,
		},
		&container.HostConfig{
			PortBindings: exposedPorts,
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		fmt.Println("Unable to create the container")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to create the container"})
		return
	}
	err = cli.ContainerStart(context.Background(), cont.ID, container.StartOptions{})
	if err != nil {
		fmt.Println("Unable to start the container")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to start the container"})
		return
	}
	fmt.Println(cont.ID)
	ports := make(map[int]int)
	for k, v := range exposedPorts {
		port, err := strconv.Atoi(v[0].HostPort)
		if err != nil {
			fmt.Println("Unable to convert the port")
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to convert the port"})
			return
		}
		ports[k.Int()] = port
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "Container with id " + cont.ID + " created and started", "id": cont.ID, "ports": ports})
}

func FindFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func BuildImage(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to create docker client"})
		return
	}
	cli.NegotiateAPIVersion(context.Background())

	var imageRequest ImageRequest
	err = c.BindJSON(&imageRequest)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	splitted := strings.Split(imageRequest.RepositoryURL, "/")
	imageName := splitted[len(splitted)-1]

	cmd := exec.Command("docker", "build", imageRequest.RepositoryURL, "-t", imageName)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Unable to build the image from: ", imageRequest.RepositoryURL)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to build the image"})
		return
	}

	fmt.Println("Image built")
}

func DeleteContainer(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to create docker client"})
		return
	}
	cli.NegotiateAPIVersion(context.Background())

	id := c.Param("id")
	StopContainer(id)

	err = cli.ContainerRemove(context.Background(), id, container.RemoveOptions{})
	if err != nil {
		fmt.Println("Unable to remove the container ", id)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Unable to remove the container"})
		return
	}
	fmt.Println("Container ", id, " removed")
}

func StopContainer(id string) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		return
	}
	cli.NegotiateAPIVersion(context.Background())

	err = cli.ContainerStop(context.Background(), id, container.StopOptions{})
	if err != nil {
		fmt.Println("Unable to stop the container ", id)
		return
	}
	fmt.Println("Container ", id, " stopped")

}

func InstantiateContainer(imageName, containerName string, portBindings []int) (map[int]int, error) {
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

func Ping(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"message": "pong"})
}
