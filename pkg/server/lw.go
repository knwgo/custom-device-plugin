package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/knwgo/custom-device-plugin/pkg/utils"
)

func (d *DPServer) list() error {
	files, err := os.ReadDir(d.devicePath)
	if err != nil {
		return err
	}

	d.m.Lock()
	defer d.m.Unlock()

	d.devices = make(map[string]utils.Device)

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fi, err := f.Info()
		if err != nil {
			klog.Errorf("get file info failed: %s", err)
			continue
		}

		inode, err := utils.GetFileInode(fi)
		if err != nil {
			klog.Errorf("get file inode failed: %s", err)
			continue
		}

		fd, err := os.ReadFile(filepath.Join(d.devicePath, fi.Name()))
		if err != nil {
			klog.Errorf("read file failed: %s", err)
			continue
		}

		dd := utils.DeviceData{}
		if err := json.Unmarshal(fd, &dd); err != nil {
			klog.Errorf("unmarshal device data failed: %s", err)
			continue
		}

		id := strconv.Itoa(int(inode))
		nn := make([]*pluginapi.NUMANode, len(dd.NumaNodes))
		for i, n := range dd.NumaNodes {
			nn[i] = &pluginapi.NUMANode{
				ID: n,
			}
		}

		d.devices[id] = utils.Device{
			Meta: utils.DeviceMeta{
				Filename: fi.Name(),
			},
			Entity: &pluginapi.Device{
				ID: id,
				Health: func() string {
					if dd.Unhealthy {
						return pluginapi.Unhealthy
					}
					return pluginapi.Healthy
				}(),
				Topology: &pluginapi.TopologyInfo{
					Nodes: nn,
				},
			},
		}
	}

	return nil
}

func (d *DPServer) watch(stop <-chan struct{}) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Errorf("create fsnotify watcher failed: %s", err)
		return err
	}

	defer func() {
		_ = fsw.Close()
	}()

	if err := fsw.Add(d.devicePath); err != nil {
		klog.Errorf("add device watcher failed: %s", err)
		return err
	}

	for {
		select {
		case event, ok := <-fsw.Events:
			if !ok {
				klog.Warning("fsnotify watcher closed event channel")
				return nil
			}
			klog.Infof("fsnotify watcher event: %s, relist", event)
			if err := d.list(); err != nil {
				klog.Errorf("relist device failed: %s", err)
				continue
			}
			d.watchCh <- struct{}{}
		case err := <-fsw.Errors:
			klog.Errorf("fsnotify watcher error: %s", err)
			return err

		case <-stop:
			klog.Info("watcher exit")
			return nil
		}
	}
}
