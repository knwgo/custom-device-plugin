package server

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/knwgo/custom-device-plugin/pkg/utils"
)

const socketName = "custom.sock"

type DPServer struct {
	srv     *grpc.Server
	devices map[string]utils.Device
	m       sync.Mutex
	watchCh chan struct{}

	kubeletSocketPath, devicePluginPath, resourceName, devicePath string
}

func NewDPServer(ksp, dpp, rn, dp string) *DPServer {
	return &DPServer{
		srv:     grpc.NewServer(),
		devices: make(map[string]utils.Device),
		m:       sync.Mutex{},
		watchCh: make(chan struct{}),

		kubeletSocketPath: ksp,
		devicePluginPath:  dpp,
		resourceName:      rn,
		devicePath:        dp,
	}
}

func (d *DPServer) Run() error {
	sockPath := d.devicePluginPath + socketName

	if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	pluginapi.RegisterDevicePluginServer(d.srv, d)

	stop := make(chan struct{})

	if err := d.list(); err != nil {
		return err
	}
	go func() {
		if err := d.watch(stop); err != nil {
			klog.Errorf("watch exit with err %v", err)
		}
	}()

	l, err := net.Listen("unix", sockPath)
	if err != nil {
		return err
	}

	go func() {
		lastCrashTime := time.Now()
		restartCount := 0

		for {
			klog.Info("start grpc server")
			if err := d.srv.Serve(l); err == nil {
				break
			}

			klog.Errorf("grpc server quit with error: %v, restart count: %d", err, restartCount)
			if restartCount > 5 {
				stop <- struct{}{}
				klog.Fatal("grpc server has repeatedly crashed recently, quiting")
			}

			timeSinceLastCrash := time.Since(lastCrashTime)
			lastCrashTime = time.Now()
			if timeSinceLastCrash.Seconds() > 3600 {
				restartCount = 1
			} else {
				restartCount++
			}
		}
	}()

	conn, err := d.dial(sockPath, time.Second*5)
	if err != nil {
		return err
	}
	_ = conn.Close()

	return nil
}

func (d *DPServer) RegisterToKubelet() error {
	conn, err := d.dial(d.kubeletSocketPath, time.Second*5)
	if err != nil {
		return err
	}

	client := pluginapi.NewRegistrationClient(conn)
	req := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     socketName,
		ResourceName: d.resourceName,
	}

	klog.Infof("register to kubelet with endpoint %s", req.Endpoint)
	if _, err = client.Register(context.Background(), req); err != nil {
		return err
	}

	return nil
}

// TODO: use grpc.NewClient
func (d *DPServer) dial(addr string, timeout time.Duration) (*grpc.ClientConn, error) {
	return grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
}
