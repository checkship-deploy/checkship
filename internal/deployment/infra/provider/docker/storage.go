package docker

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
)

const (
	// versitygw, the S3 gateway checkship runs for backups (Apache-2.0). Its posix
	// backend keeps every object as a plain file, so archives stay readable even
	// without the gateway.
	BuiltinStorageImage = "versity/versitygw:v1.11.0"
	StorageAlias        = "checkship-storage" // Name on the target's network
	StorageSubdomain    = "s3"                // Route on the target's proxy: s3.<target host>
	storagePort         = 7070
	storageService      = "storage"
	storagePortAttempts = 20 // Host ports tried, from storagePort up, when another program holds one
)

func storageProjectName(id domain.TargetID) string {
	return "checkship-internal-storage-" + strings.ToLower(string(id))
}

func (d *docker) SetupStorage(ctx context.Context, target domain.Target, spec domain.BuiltinStorageSpec) ([]domain.StorageEndpoint, error) {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return nil, err
	}

	defer client.Close()

	project, endpoints := storageProject(target, spec)
	data, ok := target.Provider().(Data)
	if !ok {
		return nil, domain.ErrInvalidProviderPayload
	}
	_, hasURL := target.Url().TryGet()
	isLocal := !data.Host.HasValue()
	needsHostEndpoint := isLocal || !hasURL

	if !needsHostEndpoint {
		return endpoints, upStorage(ctx, client, project)
	}

	// When checkship cannot rely on a shared network or target proxy, publish the
	// storage on a host port. Reuse its existing port, or pick the first one free.
	hostIP, address := storagePublishAddress(client.api.DaemonHost())
	if !isLocal {
		address = string(data.Host.MustGet())
		hostIP = ""
	}
	port, taken, err := storageHostPort(ctx, client, project.Name)

	if err != nil {
		return nil, err
	}

	for attempt := 0; ; attempt++ {
		publishStorage(project, hostIP, port)

		err = upStorage(ctx, client, project)

		if err == nil || !isPortTaken(err) || attempt+1 >= storagePortAttempts {
			break
		}

		// Held by something else than a container (or a container not running yet)
		taken[port] = true
		var available bool
		port, available = nextFreePort(port+1, taken)

		if !available {
			return nil, fmt.Errorf("no available host port for built-in backup storage")
		}
	}

	if err != nil {
		return nil, err
	}

	// Checkship in a container on this daemon joins the storage network, so its
	// alias works even when Checkship is not exposed on the target. This also
	// restores the connection on each startup after a container recreation.
	if isLocal && runsInContainer() && localDockerSocket(client.api.DaemonHost()) {
		if err = joinStorageNetwork(ctx, client, project.Networks["default"].Name); err != nil {
			return nil, err
		}
	}

	return append(endpoints, domain.StorageEndpoint{Address: net.JoinHostPort(address, strconv.Itoa(port))}), nil
}

func upStorage(ctx context.Context, client *client, project *types.Project) error {
	return client.compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			QuietPull:     true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	})
}

// Where the published port listens and the address checkship reaches it on.
// A daemon reached over TCP/SSH (for example, DOCKER_HOST=tcp://host:2375) runs
// on another machine than checkship: the port listens on every interface of that
// machine, and is reached on its host name. Otherwise only on 127.0.0.1.
func storagePublishAddress(daemonHost string) (hostIP string, address string) {
	u, err := url.Parse(daemonHost)

	if err != nil || (u.Scheme != "tcp" && u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ssh") {
		return "127.0.0.1", "127.0.0.1"
	}

	switch host := u.Hostname(); host {
	case "", "localhost", "127.0.0.1", "::1":
		return "127.0.0.1", "127.0.0.1"
	default:
		return "", host
	}
}

// The host port the storage of this project is published on if it runs, else
// the first one, from storagePort up, no running container holds. Also returns
// the ports seen held.
func storageHostPort(ctx context.Context, client *client, project string) (int, map[int]bool, error) {
	taken := make(map[int]bool)
	containers, err := client.api.ContainerList(ctx, container.ListOptions{All: true})

	if err != nil {
		return storagePort, taken, nil
	}

	for _, c := range containers {
		for _, p := range c.Ports {
			if p.PublicPort == 0 {
				continue
			}

			if c.Labels[api.ProjectLabel] == project && c.Labels[api.ServiceLabel] == storageService &&
				int(p.PrivatePort) == storagePort && c.State == "running" {
				return int(p.PublicPort), taken, nil
			}

			taken[int(p.PublicPort)] = true
		}
	}

	port, available := nextFreePort(storagePort, taken)
	if !available {
		return 0, taken, fmt.Errorf("no available host port for built-in backup storage")
	}
	return port, taken, nil
}

func nextFreePort(from int, taken map[int]bool) (int, bool) {
	port := from

	for port <= 65535 && taken[port] {
		port++
	}

	return port, port <= 65535
}

func publishStorage(project *types.Project, hostIP string, port int) {
	service := project.Services[storageService]
	service.Ports = []types.ServicePortConfig{
		{Target: storagePort, Published: strconv.Itoa(port), HostIP: hostIP, Protocol: "tcp"},
	}
	project.Services[storageService] = service
}

func isPortTaken(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "port is already allocated") || strings.Contains(msg, "address already in use")
}

// Connects the Checkship container to the local storage network. The caller
// only invokes this for a local socket-backed daemon.
func joinStorageNetwork(ctx context.Context, client *client, network string) error {
	hostname, err := os.Hostname()

	if err != nil {
		return fmt.Errorf("could not identify the Checkship container: %w", err)
	}
	if hostname == "" {
		return fmt.Errorf("could not identify the Checkship container: hostname is empty")
	}

	self, err := client.api.ContainerInspect(ctx, hostname)

	// The host name of a container is its short ID, unless set otherwise: make
	// sure the container found is this one and not another with that name
	if err != nil {
		return fmt.Errorf("could not find the Checkship container on the local Docker daemon: %w", err)
	}

	if self.Config == nil || (!strings.HasPrefix(self.ID, hostname) && self.Config.Hostname != hostname) {
		return fmt.Errorf("could not match the Checkship container on the local Docker daemon")
	}

	if self.NetworkSettings != nil {
		if _, isOn := self.NetworkSettings.Networks[network]; isOn {
			return nil
		}
	}

	if err = client.api.NetworkConnect(ctx, network, self.ID, nil); err != nil &&
		!errdefs.IsForbidden(err) && !errdefs.IsConflict(err) {
		return fmt.Errorf("could not connect the Checkship container to the built-in storage network: %w", err)
	}

	return nil
}

func localDockerSocket(daemonHost string) bool {
	u, err := url.Parse(daemonHost)
	return err == nil && (u.Scheme == "unix" || u.Scheme == "npipe")
}

func runsInContainer() bool {
	for _, f := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}

	return false
}

func (d *docker) RemoveStorage(ctx context.Context, target domain.Target) error {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	return client.compose.Down(ctx, storageProjectName(target.ID()), api.DownOptions{
		RemoveOrphans: true,
		Volumes:       true,
	})
}

// Compose project of the built-in storage, and the ways to reach it, best first:
//   - its alias on the target's network, when checkship's own container is on
//     that network (exposed on the target, or joined by SetupStorage);
//   - s3.<host> through the target's proxy, when the target has a URL.
//
// When Checkship cannot share a Docker network with the target, SetupStorage
// publishes a host port as a fallback and adds it last.
func storageProject(target domain.Target, spec domain.BuiltinStorageSpec) (*types.Project, []domain.StorageEndpoint) {
	name := storageProjectName(target.ID())
	idLower := domain.TargetID(strings.ToLower(string(target.ID())))
	labels := types.Labels{TargetLabel: string(target.ID())}
	image := spec.Image

	if image == "" {
		image = BuiltinStorageImage
	}

	service := types.ServiceConfig{
		Name:    storageService,
		Image:   image,
		Restart: types.RestartPolicyUnlessStopped,
		Command: types.ShellCommand{"--port", ":" + strconv.Itoa(storagePort), "posix", "/data"},
		Environment: types.MappingWithEquals{
			"ROOT_ACCESS_KEY": &spec.AccessKey,
			"ROOT_SECRET_KEY": &spec.SecretKey,
		},
		Labels: labels,
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeVolume, Source: "data", Target: "/data"},
		},
		Networks: map[string]*types.ServiceNetworkConfig{
			"default": {Aliases: []string{StorageAlias}},
		},
		CustomLabels: getProjectCustomLabels(name, storageService, ""),
	}

	project := &types.Project{
		Name: name,
		Networks: types.Networks{
			"default": types.NetworkConfig{
				Name:   name + "_network",
				Labels: labels,
			},
		},
		Volumes: types.Volumes{
			"data": types.VolumeConfig{
				Name:   name + "_data",
				Labels: types.Labels{StorageLabel: string(target.ID())},
			},
		},
	}

	endpoints := []domain.StorageEndpoint{
		{Address: StorageAlias + ":" + strconv.Itoa(storagePort)},
	}

	// A target with a URL runs a proxy on its public network, which checkship's
	// own container joins when it is exposed there: the storage joins it too.
	if url, hasUrl := target.Url().TryGet(); hasUrl {
		project.Networks["default"] = types.NetworkConfig{
			Name:     targetPublicNetworkName(idLower),
			External: true,
		}

		// Picked up by the target's proxy, which routes s3.<host> to the storage
		service.Labels = appendLabels(types.Labels{
			SubdomainLabel: StorageSubdomain,
			"traefik.http.services." + name + ".loadbalancer.server.port": strconv.Itoa(storagePort),
		}, labels)
		endpoints = append(endpoints, domain.StorageEndpoint{
			Address: StorageSubdomain + "." + url.Host(),
			UseSSL:  url.UseSSL(),
		})
	}

	project.Services = types.Services{storageService: service}

	return project, endpoints
}
