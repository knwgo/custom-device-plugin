package utils

type DeviceData struct {
	NumaNodes []int64 `json:"Nodes"`
	Unhealthy bool    `json:"Unhealthy"`
}
