/*
	this client is based on docker-go-client

*/

package dockerClient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"github.com/mitchellh/go-homedir"
	"io"
	kube_proxy "minik8s/configs"
	"minik8s/pkg/api/core"
	"minik8s/pkg/kubelet/config"
	"os"
)

func GetNewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.WithVersion("1.41"))
}

// GetAllContainers get all containers, including running / stopped
// docker ps
func GetAllContainers() ([]types.Container, error) {
	cli, err := GetNewClient()
	if err != nil {
		return nil, err
	}
	return cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
}

// GetRunningContainers get running containers
// docker ps -a
func GetRunningContainers() ([]types.Container, error) {
	cli, err := GetNewClient()
	if err != nil {
		return nil, err
	}
	return cli.ContainerList(context.Background(), types.ContainerListOptions{})
}

func GetAllImages() ([]types.ImageSummary, error) {
	cli, err := GetNewClient()
	if err != nil {
		return nil, err
	}
	return cli.ImageList(context.Background(), types.ImageListOptions{All: true})
}

// ImageInSet judge if image is in a set
// return true if in,
func ImageInSet(image string, imageSet []string) bool {
	fullName := image + ":latest" // default is latest
	for _, name := range imageSet {
		if name == image {
			return true
		}
		if fullName == name {
			return true
		}
	}
	return false
}

// PullImages  pull some images in the array
// ignore the images which already exists in local
func PullImages(targets []string) error {
	historyImages, err := GetAllImages()
	if err != nil {
		return err
	}
	var filteredTarget []string
	m := make(map[string]int)
	for _, image := range targets {
		var flag bool
		for _, historyImage := range historyImages {
			if ImageInSet(image, historyImage.RepoTags) {
				flag = true
				break
			}
		}
		_, ok := m[image]
		if ok == true {
			continue
		}
		if flag == false {
			filteredTarget = append(filteredTarget, image)
			fmt.Printf("Image doesn't exist: %v\n", image)
			m[image] = 1
		}
	}
	if filteredTarget != nil {
		for _, target := range filteredTarget {
			err := PullOneImage(target)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PullOneImage 拉取一个镜像
// ImagePull自带重试功能
// 如果没有找到，会报panic
func PullOneImage(target string) error {
	print("start to pull image: ", target)
	cli, err := GetNewClient()
	if err != nil {
		return err
	}
	out, err := cli.ImagePull(context.Background(), target, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	// 延迟返回，在这个pullOneImage函数完全结束之后延迟返回
	defer out.Close()
	io.Copy(io.Discard, out)
	return nil
}

// 根据containerId start一个已有的container
// 有些容器可能无法使用
func StartOneContainer(id string) error {
	cli, err := GetNewClient()
	if err != nil {
		return err
	}
	err = cli.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

// 如果容器没有启动，等价于start
func RestartContainer(id string) error {
	cli, err := GetNewClient()
	if err != nil {
		return err
	}
	err = cli.ContainerRestart(context.Background(), id, container.StopOptions{})
	if err != nil {
		return err
	}
	return nil
}

// KillContainer 向id-container发送一个signal
// 比如SIGKILL, SIGINT
func KillContainer(id string, signal string) error {
	cli, err := GetNewClient()
	if err != nil {
		return err
	}
	err = cli.ContainerKill(context.Background(), id, signal)
	if err != nil {
		return err
	}
	return nil
}

func StopContainer(id string) error {
	cli, err := GetNewClient()
	if err != nil {
		return err
	}
	err = cli.ContainerStop(context.Background(), id, container.StopOptions{})
	return err
}

func DeleteContainer(id string) error {
	cli, err := GetNewClient()
	if err != nil {
		return err
	}
	_, err = cli.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil
	}
	err = cli.ContainerStop(context.Background(), id, container.StopOptions{})
	if err != nil {
		return err
	}
	err = cli.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

func DeleteContainers(ids []string) error {
	for _, name := range ids {
		err := DeleteContainer(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetContainerInfo(id string) (types.ContainerJSON, error) {
	cli, err := GetNewClient()
	if err != nil {
		return types.ContainerJSON{}, err
	}
	info, err := cli.ContainerInspect(context.Background(), id)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return info, nil
}

// 如果已经存在volume， 不管
// 注意
func CreateVolume(name string, hostpath *string) (volume.Volume, error) {
	cli, err := GetNewClient()
	if err != nil {
		return volume.Volume{}, err
	}
	mapOptions := map[string]string{}
	if hostpath != nil {
		mapOptions["type"] = "local"
		mapOptions["o"] = "bind"
		mapOptions["device"] = *hostpath
	}
	resp, err := cli.VolumeCreate(context.Background(), volume.CreateOptions{
		Name:       name,
		Driver:     config.DEFAULT_DRIVER,
		DriverOpts: mapOptions,
	})
	if err != nil {
		return volume.Volume{}, err
	}
	return resp, nil
}

/*
获取容器的网络设置
如果用在pause容器中，可以获取pod的网络设置
*/
func GetNetworkSettings(id string) (*types.NetworkSettings, error) {
	cli, err := GetNewClient()
	if err != nil {
		return nil, err
	}
	res, err := cli.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return res.NetworkSettings, nil
}

// CreateContainer deprecated
func CreateContainer(con core.Container) (container.CreateResponse, error) {
	cli, err := GetNewClient()
	if err != nil {
		return container.CreateResponse{}, err
	}
	err = PullImages([]string{con.Image})
	if err != nil {
		return container.CreateResponse{}, err
	}
	var mountsInfo []mount.Mount
	if con.VolumeMounts != nil {
		for _, volume := range con.VolumeMounts {
			mountsInfo = append(mountsInfo, mount.Mount{
				Type:   mount.TypeVolume,
				Source: volume.Name,
				Target: volume.MountPath,
			})
		}
	}

	resp, err := cli.ContainerCreate(context.Background(),
		&container.Config{Image: con.Image, Entrypoint: con.EntryPoint, Tty: con.Tty},
		&container.HostConfig{
			Mounts: mountsInfo,
			DNS:    []string{config.DnsAddress},
		}, nil, nil, con.Name)
	return resp, err
}

func CreatePauseContainer(name string, ports []core.Port) (container.CreateResponse, error) {
	cli, err := GetNewClient()
	if err != nil {
		return container.CreateResponse{}, err
	}
	// this for outside-call
	err = PullImages([]string{config.PAUSE_IMAGE_NAME})
	if err != nil {
		return container.CreateResponse{}, err
	}
	var portSet nat.PortSet
	portSet = make(nat.PortSet, len(ports))
	for _, port := range ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}
		if port.Protocol == "tcp" {
			res, err := nat.NewPort("tcp", port.PortNumber)
			if err != nil {
				return container.CreateResponse{}, err
			}
			portSet[res] = struct{}{}
			continue
		}
		if port.Protocol == "udp" {
			res, err := nat.NewPort("udp", port.PortNumber)
			if err != nil {
				return container.CreateResponse{}, err
			}
			portSet[res] = struct{}{}
			continue
		}
		fmt.Println("[dockerClient]:" + "unsupported network protocol")
	}
	response, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image:        config.PAUSE_IMAGE_NAME,
		ExposedPorts: portSet,
	}, &container.HostConfig{
		IpcMode: container.IPCModeShareable,
		DNS:     []string{config.DnsAddress},
	}, nil, nil, name)
	return response, err
}

func CreateCoreDNSPod(pod core.Pod) ([]core.ContainerMeta, *types.NetworkSettings, error) {
	containers := pod.Spec.Containers

	cli, err := GetNewClient()
	if err != nil {
		return nil, nil, err
	}
	var totalPorts []core.Port
	var res []core.ContainerMeta
	var images []string
	var names []string
	for _, v := range containers {
		names = append(names, v.Name)
		images = append(images, v.Image)
		for _, port := range v.Ports {
			totalPorts = append(totalPorts, port)
		}
	}
	// 首先删除之前同名的container
	err = DeleteContainers(names)
	if err != nil {
		return nil, nil, err
	}
	// 拉取镜像列表
	err = PullImages(images)
	if err != nil {
		return nil, nil, err
	}

	// 创建Volumes并mount hostpath
	for _, vo := range pod.Spec.Volumes {
		if vo.HostPath != "" { // mountHost
			_, err := CreateVolume(vo.Name, &vo.HostPath)
			if err != nil {
				fmt.Println(err.Error())
				return nil, nil, err
			}
		} else { // not mountHost
			_, err := CreateVolume(vo.Name, nil)
			if err != nil {
				fmt.Println(err.Error())
				return nil, nil, err
			}
		}
	}
	var onlyID string
	for _, v := range containers {
		// 配置容器的挂载
		var mountsInfo []mount.Mount
		if v.VolumeMounts != nil {
			for _, volume := range v.VolumeMounts {
				mountsInfo = append(mountsInfo, mount.Mount{
					Type:   mount.TypeVolume,
					Source: volume.Name,
					Target: volume.MountPath,
				})
			}
		}
		// 创建容器
		resp, err := cli.ContainerCreate(context.Background(), &container.Config{
			Image:      v.Image,
			Entrypoint: v.EntryPoint,
			Cmd:        v.Command,
			Tty:        v.Tty,
			//Env:        []string{"PATH=$PATH:/bin:/tmp/host_path"},
		}, &container.HostConfig{
			Mounts: mountsInfo,
		}, nil, nil, v.Name)
		onlyID = resp.ID
		if err != nil {
			return nil, nil, err
		}
		res = append(res, core.ContainerMeta{Name: v.Name, Id: resp.ID})
	}

	// 启动
	for _, v := range res {
		err := cli.ContainerStart(context.Background(), v.Id, types.ContainerStartOptions{})
		if err != nil {
			return nil, nil, err
		}
	}
	netSetting, err := GetNetworkSettings(onlyID)
	if err != nil {
		return nil, nil, err
	}
	return res, netSetting, nil
}

// createNormalPod, not coreDNS pod
func CreatePod(pod core.Pod) ([]core.ContainerMeta, *types.NetworkSettings, error) {
	containers := pod.Spec.Containers

	cli, err := GetNewClient()
	if err != nil {
		return nil, nil, err
	}
	var totalPorts []core.Port
	var res []core.ContainerMeta
	images := []string{config.PAUSE_IMAGE_NAME}
	names := []string{config.PAUSE_NAME}
	curPauseName := pod.Name + "-" + config.PAUSE_NAME
	for _, v := range containers {
		names = append(names, v.Name)
		images = append(images, v.Image)
		for _, port := range v.Ports {
			totalPorts = append(totalPorts, port)
		}
	}
	// 首先删除之前同名的container
	err = DeleteContainers(names)
	if err != nil {
		return nil, nil, err
	}
	err = DeleteContainer(curPauseName)
	if err != nil {
		return nil, nil, err
	}
	// 拉取镜像列表
	err = PullImages(images)
	if err != nil {
		return nil, nil, err
	}
	// 创建pause
	pauseContainer, err := CreatePauseContainer(curPauseName, totalPorts)
	if err != nil {
		return nil, nil, err
	}
	curPauseID := pauseContainer.ID
	res = append(res, core.ContainerMeta{Name: curPauseName, Id: curPauseID})

	// 创建Volumes并mount hostpath
	for _, vo := range pod.Spec.Volumes {
		if vo.HostPath != "" { // mountHost
			_, err := CreateVolume(vo.Name, &vo.HostPath)
			if err != nil {
				fmt.Println(err.Error())
				return nil, nil, err
			}
		} else { // not mountHost
			_, err := CreateVolume(vo.Name, nil)
			if err != nil {
				fmt.Println(err.Error())
				return nil, nil, err
			}
		}
	}

	for _, v := range containers {
		// 配置容器的挂载
		var mountsInfo []mount.Mount
		if v.VolumeMounts != nil {
			for _, volume := range v.VolumeMounts {
				mountsInfo = append(mountsInfo, mount.Mount{
					Type:   mount.TypeVolume,
					Source: volume.Name,
					Target: volume.MountPath,
				})
			}
		}
		// 创建容器
		resp, err := cli.ContainerCreate(context.Background(), &container.Config{
			Image:      v.Image,
			Entrypoint: v.EntryPoint,
			Cmd:        v.Command,
			Tty:        v.Tty,
			//Env:        []string{"PATH=$PATH:/bin:/tmp/host_path"},
		}, &container.HostConfig{
			NetworkMode: container.NetworkMode("container:" + curPauseID),
			IpcMode:     container.IpcMode("container:" + curPauseID),
			PidMode:     container.PidMode("container:" + curPauseID),
			Mounts:      mountsInfo,
		}, nil, nil, v.Name)
		if err != nil {
			return nil, nil, err
		}
		res = append(res, core.ContainerMeta{Name: v.Name, Id: resp.ID})
	}

	// 启动
	for _, v := range res {
		err := cli.ContainerStart(context.Background(), v.Id, types.ContainerStartOptions{})
		if err != nil {
			return nil, nil, err
		}
	}
	netSetting, err := GetNetworkSettings(curPauseID)
	if err != nil {
		return nil, nil, err
	}
	return res, netSetting, nil
}

func DeletePod(pod core.Pod) error {
	containers := pod.Spec.Containers

	var names []string
	curPauseName := pod.Name + "-" + config.PAUSE_NAME
	for _, v := range containers {
		names = append(names, v.Name)
	}
	if pod.ObjectMeta.Name != kube_proxy.CoreDNSPodName {
		names = append(names, curPauseName)
	}
	err := DeleteContainers(names)
	return err
}

func DockerStats() (*types.ContainerStats, error) {
	cli, err := GetNewClient()
	if err != nil {
		return nil, err
	}
	resp, err := cli.ContainerStats(context.Background(), "Etcd-server", true)
	j, err := json.Marshal(resp.Body)
	fmt.Println(j)
	fmt.Println(resp.Body)
	fmt.Println(resp.OSType)
	return nil, nil
}

func GetDockerStats(name string) (types.ContainerStats, error) {
	containers, err := GetAllContainers()
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, v := range containers {
		if v.Names[0] == "/"+name {
			fmt.Println(v.ID)
			cli, err := GetNewClient()
			if err != nil {
				fmt.Println(err.Error())
			}
			resp, err := cli.ContainerStats(context.Background(), v.ID, false)
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println(resp)
			return resp, nil
		}
	}
	return types.ContainerStats{}, fmt.Errorf("no such container")
}

func GetContext(filePath string) io.Reader {
	// Use homedir.Expand to resolve paths like '~/repos/myrepo'
	filePath, _ = homedir.Expand(filePath)
	ctx, _ := archive.TarWithOptions(filePath, &archive.TarOptions{})
	return ctx
}

func ImageBuild(buildPath string, image string) error {
	ctx := context.Background()
	cli, err := GetNewClient()
	if err != nil {
		fmt.Println("Docker client init failed.", err)
		return err
	}

	// 准备构建参数
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{image},
	}

	// 发起构建请求
	buildResponse, err := cli.ImageBuild(ctx, GetContext(buildPath), buildOptions)
	if err != nil {
		fmt.Println("Docker build failed.", err)
		return err
	}
	defer buildResponse.Body.Close()

	bodyBytes, err := io.ReadAll(buildResponse.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	bodyString := string(bodyBytes)
	fmt.Println(bodyString)

	fmt.Println("Docker build completed.")
	return nil
}

func ImagePush(image string) error {
	ctx := context.Background()
	cli, err := GetNewClient()
	if err != nil {
		fmt.Println("Docker client init failed.", err)
		return err
	}

	var authConfig = types.AuthConfig{
		Username: "luhaoqi",
		Password: os.Getenv("DOCKER_TOKEN"),
	}
	authConfigBytes, _ := json.Marshal(authConfig)
	authConfigEncoded := base64.URLEncoding.EncodeToString(authConfigBytes)

	opts := types.ImagePushOptions{RegistryAuth: authConfigEncoded}

	buildResponse, err := cli.ImagePush(ctx, image, opts)
	if err != nil {
		return err
	}

	defer buildResponse.Close()

	bodyBytes, err := io.ReadAll(buildResponse)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	bodyString := string(bodyBytes)
	fmt.Println(bodyString)
	fmt.Println("Docker push completed.")
	return nil
}

func ImageRemove(name string) error {
	return nil
}
