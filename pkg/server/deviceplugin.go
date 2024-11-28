package server

import (
	"context"
	"strings"

	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func (d *DPServer) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	klog.Info("GetDevicePluginOptions called")
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                true,
		GetPreferredAllocationAvailable: false,
	}, nil
}

func (d *DPServer) ListAndWatch(_ *pluginapi.Empty, server pluginapi.DevicePlugin_ListAndWatchServer) error {
	klog.Info("ListAndWatch called")

	send := func() error {
		d.m.Lock()
		devs := make([]*pluginapi.Device, len(d.devices))
		i := 0
		for _, dev := range d.devices {
			devs[i] = dev
			i++
		}
		d.m.Unlock()

		if err := server.Send(&pluginapi.ListAndWatchResponse{
			Devices: devs,
		}); err != nil {
			return err
		}

		return nil
	}

	if err := send(); err != nil {
		return err
	}

	for range d.watchCh {
		if err := send(); err != nil {
			return err
		}
		klog.Info("device list resent")
	}

	return nil
}

func (d *DPServer) GetPreferredAllocation(_ context.Context, request *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	klog.Info("GetPreferredAllocation called")
	return nil, nil
}

func (d *DPServer) Allocate(_ context.Context, request *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	klog.Info("Allocate called")
	resp := &pluginapi.AllocateResponse{}
	for _, req := range request.ContainerRequests {
		klog.Infof("received request: %v", strings.Join(req.DevicesIDs, ","))
		cr := pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				"CUSTOM_DEVICES": strings.Join(req.DevicesIDs, ","),
			},
		}

		resp.ContainerResponses = append(resp.ContainerResponses, &cr)
	}
	return resp, nil
}

func (d *DPServer) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	klog.Info("PreStartContainer called")
	return &pluginapi.PreStartContainerResponse{}, nil
}
