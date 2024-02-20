package handlers

import (
	tar2 "archive/tar"
	"bytes"
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

type ContainerRequest struct {
	ImageName     string `json:"imageName"`
	ContainerName string `json:"containerName"`
	Port          string `json:"port"`
}

type ImageRequest struct {
	RepositoryURL string `json:"repository"`
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

func GetAllContainersFromVM(c *gin.Context) {
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
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		panic(err)
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
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())

	// To set a specific image version, use the following format: imageName:version
	var containerRequest ContainerRequest
	err = c.BindJSON(&containerRequest)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// check if the image exists
	_, _, err = cli.ImageInspectWithRaw(context.Background(), containerRequest.ImageName)
	if err != nil {
		// pull the image if it doesn't exist
		reader, err := cli.ImagePull(context.Background(), containerRequest.ImageName, types.ImagePullOptions{})
		if err != nil {
			fmt.Println("Unable to pull the image")
			panic(err)
		}
		defer reader.Close()
		_, err = io.ReadAll(reader)
		if err != nil {
			panic(err)
		}
		_, _, err = cli.ImageInspectWithRaw(context.Background(), containerRequest.ImageName)
		if err != nil {
			fmt.Println("Unable to inspect the image")
			panic(err)
		}
	}

	var portBinding nat.Port
	portBinding, err = nat.NewPort("tcp", containerRequest.Port)

	// Create a container
	cont, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: containerRequest.ImageName,
			ExposedPorts: nat.PortSet{
				portBinding: struct{}{},
			},
		},
		&container.HostConfig{},
		nil,
		nil,
		containerRequest.ContainerName,
	)
	if err != nil {
		fmt.Println("Unable to create the container")
		panic(err)
	}
	err = cli.ContainerStart(context.Background(), cont.ID, container.StartOptions{})
	if err != nil {
		fmt.Println("Unable to start the container")
		panic(err)
	}
}

func BuildImage(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())

	var imageRequest ImageRequest
	err = c.BindJSON(&imageRequest)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Create a tar file with the Dockerfile
	var buf bytes.Buffer
	tarWriter := tar2.NewWriter(&buf)

	contents := `FROM nginx
		COPY . /usr/share/nginx/html
		CMD ["nginx", "-g", "daemon off;"]
		`

	header := &tar2.Header{
		Name:     "Dockerfile",
		Mode:     0777,
		Size:     int64(len(contents)),
		Typeflag: tar2.TypeReg,
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		fmt.Println("Unable to write the header")
		panic(err)
	}

	if _, err := tarWriter.Write([]byte(contents)); err != nil {
		fmt.Println("Unable to write the content")
		panic(err)
	}

	if err := tarWriter.Close(); err != nil {
		fmt.Println("Unable to close the writer")
		panic(err)
	}

	reader := bytes.NewReader(buf.Bytes())

	BuildOptions := types.ImageBuildOptions{
		Context:    reader,
		Dockerfile: "Dockerfile",
		Tags:       []string{"nosql:1.0.0"},
	}

	resp, err := cli.ImageBuild(context.Background(), reader, BuildOptions)
	if err != nil {
		fmt.Println("Unable to build the image")
		panic(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Unable to read the response")
		panic(err)
	}
	fmt.Println(string(body))

	fmt.Println("Image built")
}

func DeleteContainer(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())

	id := c.Param("id")
	StopContainer(id)

	err = cli.ContainerRemove(context.Background(), id, container.RemoveOptions{})
	if err != nil {
		fmt.Println("Unable to remove the container ", id)
		panic(err)
	}
	fmt.Println("Container ", id, " removed")
}

func StopContainer(id string) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Unable to create docker client")
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())

	err = cli.ContainerStop(context.Background(), id, container.StopOptions{})
	if err != nil {
		fmt.Println("Unable to stop the container ", id)
		panic(err)
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
