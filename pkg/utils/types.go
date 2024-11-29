package utils

import pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

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
