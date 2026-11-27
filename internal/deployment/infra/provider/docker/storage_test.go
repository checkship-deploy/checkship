package docker

import (
	"errors"
	"strings"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func testLocalTarget(t *testing.T) domain.Target {
	t.Helper()
	target, err := domain.NewTarget("local", domain.NewProviderConfigRequirement(Data{}, true), "test-user")
	assert.Nil(t, err)
	return target
}

func Test_StoragePublishAddress(t *testing.T) {
	tests := []struct {
		name       string
		daemonHost string
		wantIP     string
		want       string
	}{
		{name: "unix socket", daemonHost: "unix:///var/run/docker.sock", wantIP: "127.0.0.1", want: "127.0.0.1"},
		{name: "local tcp", daemonHost: "tcp://localhost:2375", wantIP: "127.0.0.1", want: "127.0.0.1"},
		{name: "remote tcp", daemonHost: "tcp://docker.example.test:2375", want: "docker.example.test"},
		{name: "remote ipv6 tcp", daemonHost: "tcp://[2001:db8::1]:2375", want: "2001:db8::1"},
		{name: "unparseable", daemonHost: "://bad", wantIP: "127.0.0.1", want: "127.0.0.1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ip, address := storagePublishAddress(test.daemonHost)
			assert.Equal(t, test.wantIP, ip)
			assert.Equal(t, test.want, address)
		})
	}
}

func Test_StorageProjectUsesBuiltinImageAndPersistentVolume(t *testing.T) {
	target := testLocalTarget(t)
	project, endpoints := storageProject(target, domain.BuiltinStorageSpec{AccessKey: "key", SecretKey: "secret"})
	service := project.Services[storageService]

	assert.Equal(t, BuiltinStorageImage, service.Image)
	assert.Equal(t, "checkship-internal-storage-"+strings.ToLower(string(target.ID()))+"_data", project.Volumes["data"].Name)
	assert.Equal(t, "/data", service.Volumes[0].Target)
	assert.Equal(t, StorageAlias+":7070", endpoints[0].Address)
	assert.Equal(t, 0, len(service.Ports)) // SetupStorage picks a collision-free host port.
}

func Test_NextFreePort(t *testing.T) {
	port, available := nextFreePort(7070, map[int]bool{7070: true, 7071: true})
	assert.True(t, available)
	assert.Equal(t, 7072, port)

	port, available = nextFreePort(65535, map[int]bool{65535: true})
	assert.False(t, available)
	assert.Equal(t, 65536, port)
}

func Test_PublishStorageAndDetectPortConflicts(t *testing.T) {
	target := testLocalTarget(t)
	project, _ := storageProject(target, domain.BuiltinStorageSpec{})
	publishStorage(project, "127.0.0.1", 7072)
	ports := project.Services[storageService].Ports

	assert.HasLength(t, 1, ports)
	assert.Equal(t, uint32(storagePort), ports[0].Target)
	assert.Equal(t, "7072", ports[0].Published)
	assert.Equal(t, "127.0.0.1", ports[0].HostIP)
	assert.True(t, isPortTaken(errors.New("port is already allocated")))
	assert.True(t, isPortTaken(errors.New("bind: address already in use")))
	assert.False(t, isPortTaken(errors.New("image pull failed")))
}

func Test_LocalDockerSocket(t *testing.T) {
	assert.True(t, localDockerSocket("unix:///var/run/docker.sock"))
	assert.True(t, localDockerSocket("npipe:////./pipe/docker_engine"))
	assert.False(t, localDockerSocket("tcp://docker.example.test:2375"))
	assert.False(t, localDockerSocket("ssh://user@docker.example.test"))
}
