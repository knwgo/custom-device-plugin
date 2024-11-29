package utils

import (
	"time"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type DeviceData struct {
	NumaNodes []int64 `json:"Nodes"`
	Unhealthy bool    `json:"Unhealthy"`
}

type DeviceMeta struct {
	Filename string `json:"Filename"`
}

type Device struct {
	Meta   DeviceMeta
	Entity *pluginapi.Device
}

type SortableDevice struct {
	ID      string
	ModTime time.Time
}

type SortableDeviceList []SortableDevice

func (s SortableDeviceList) Len() int      { return len(s) }
func (s SortableDeviceList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortableDeviceList) Less(i, j int) bool {
	return s[i].ModTime.Before(s[j].ModTime)
}
