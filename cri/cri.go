package cri

import (
	"context"
	"os"
	"time"

	log "github.com/golang/glog"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	containerdPath     = "/var/run/containerd/containerd.sock"
	condSocketPath     = "unix://" + containerdPath
	dockerSocketPath   = "unix:///var/run/dockershim.sock"
	defaultDialTimeout = 10 * time.Second
)

type CRIAPIs interface {
	GetReadyPodSandboxes() ([]*SandboxInfo, error)
}

type SandboxInfo struct {
	ContainerId string
	PodName     string
	NameSpace   string
}

type CRIClient struct {
	socketPath string
}

func New() *CRIClient {
	c := &CRIClient{}

	c.socketPath = dockerSocketPath
	if info, err := os.Stat(containerdPath); err == nil && !info.IsDir() {
		log.Infof("conatinerd socket %v exists, use it to connect to containerd", condSocketPath)
		c.socketPath = condSocketPath
	} else {
		log.Infof("use docker shim socket %v to connect to docker", dockerSocketPath)
	}

	return c
}

//GetReadyPodSandboxes get ready sandboxIDs
func (c *CRIClient) GetReadyPodSandboxes() ([]*SandboxInfo, error) {
	ctx := context.TODO()
	log.Infof("Getting ready pod sandboxes from %q", c.socketPath)
	conn, err := grpc.Dial(c.socketPath, grpc.WithInsecure(), grpc.WithNoProxy(),
		grpc.WithBlock(), grpc.WithTimeout(defaultDialTimeout))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := runtimeapi.NewRuntimeServiceClient(conn)

	// List all ready sandboxes from the CRI
	sandboxes, err := client.ListPodSandbox(ctx, &runtimeapi.ListPodSandboxRequest{
		Filter: &runtimeapi.PodSandboxFilter{
			State: &runtimeapi.PodSandboxStateValue{
				State: runtimeapi.PodSandboxState_SANDBOX_READY,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sandboxInfos := make([]*SandboxInfo, 0, len(sandboxes.GetItems()))
	for _, sandbox := range sandboxes.GetItems() {
		info := SandboxInfo{
			ContainerId: sandbox.Id,
			PodName:     sandbox.Metadata.Name,
			NameSpace:   sandbox.Metadata.Namespace,
		}
		sandboxInfos = append(sandboxInfos, &info)
	}
	return sandboxInfos, nil
}
