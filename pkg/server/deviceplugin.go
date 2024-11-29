package server

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/knwgo/custom-device-plugin/pkg/utils"
)

func (d *DPServer) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	klog.Info("GetDevicePluginOptions called")
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                true,
		GetPreferredAllocationAvailable: true,
	}, nil
}

func (d *DPServer) ListAndWatch(_ *pluginapi.Empty, server pluginapi.DevicePlugin_ListAndWatchServer) error {
	klog.Info("ListAndWatch called")

	send := func() error {
		d.m.Lock()
		devs := make([]*pluginapi.Device, len(d.devices))
		i := 0
		for _, dev := range d.devices {
			devs[i] = dev.Entity
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

	resp := &pluginapi.PreferredAllocationResponse{ContainerResponses: make([]*pluginapi.ContainerPreferredAllocationResponse, len(request.ContainerRequests))}

	getPreferred := func(allocationRequest *pluginapi.ContainerPreferredAllocationRequest) (*pluginapi.ContainerPreferredAllocationResponse, error) {
		r := &pluginapi.ContainerPreferredAllocationResponse{DeviceIDs: make([]string, 0)}
		r.DeviceIDs = append(r.DeviceIDs, allocationRequest.MustIncludeDeviceIDs...)
		mustMap := make(map[string]struct{})
		for i := range allocationRequest.MustIncludeDeviceIDs {
			mustMap[allocationRequest.MustIncludeDeviceIDs[i]] = struct{}{}
		}

		d.m.Lock()
		defer d.m.Unlock()

		sd := make(utils.SortableDeviceList, len(allocationRequest.AvailableDeviceIDs))
		for i, id := range allocationRequest.AvailableDeviceIDs {
			dev, exist := d.devices[id]
			if !exist {
				return nil, status.Errorf(codes.InvalidArgument, "device %s not found", id)
			}

			f, err := os.Open(filepath.Join(d.devicePath, dev.Meta.Filename))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to open file: %s", filepath.Join(d.devicePath, dev.Meta.Filename))
			}

			fi, err := f.Stat()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get file %s stat", filepath.Join(d.devicePath, dev.Meta.Filename))
			}

			sd[i] = utils.SortableDevice{
				ID:      id,
				ModTime: fi.ModTime(),
			}
		}

		sort.Sort(sd)
		amount := int(allocationRequest.AllocationSize) - len(allocationRequest.MustIncludeDeviceIDs)
		for _, dev := range sd {
			if amount <= 0 {
				break
			}

			if _, ok := mustMap[dev.ID]; ok {
				continue
			}

			r.DeviceIDs = append(r.DeviceIDs, dev.ID)
			amount--
		}

		if amount > 0 {
			return nil, status.Errorf(codes.InvalidArgument, "len AvailableDeviceIDs: %d, len MustIncludeDeviceIDs: %d, AllocationSize: %d",
				len(allocationRequest.AvailableDeviceIDs), len(allocationRequest.MustIncludeDeviceIDs), allocationRequest.AllocationSize)
		}

		return r, nil
	}

	for i, req := range request.ContainerRequests {
		res, err := getPreferred(req)
		if err != nil {
			return nil, err
		}
		resp.ContainerResponses[i] = res
	}

	return resp, nil
}

func (d *DPServer) Allocate(_ context.Context, request *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	klog.Info("Allocate called")

	d.m.Lock()
	defer d.m.Unlock()

	resp := &pluginapi.AllocateResponse{}
	for _, req := range request.ContainerRequests {
		klog.Infof("received request: %v", strings.Join(req.DevicesIDs, ","))
		deviceMount := make([]*pluginapi.Mount, len(req.DevicesIDs))
		for i, id := range req.DevicesIDs {
			dev, exist := d.devices[id]
			if !exist {
				return nil, status.Errorf(codes.InvalidArgument, "device %s not found", id)
			}
			devPath := filepath.Join(d.devicePath, dev.Meta.Filename)
			deviceMount[i] = &pluginapi.Mount{
				ContainerPath: devPath,
				HostPath:      devPath,
				ReadOnly:      true,
			}
		}

		cr := pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				"CUSTOM_DEVICES": strings.Join(req.DevicesIDs, ","),
			},
			Mounts: deviceMount,
		}

		resp.ContainerResponses = append(resp.ContainerResponses, &cr)
	}
	return resp, nil
}

func (d *DPServer) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	klog.Info("PreStartContainer called")
	return &pluginapi.PreStartContainerResponse{}, nil
}
