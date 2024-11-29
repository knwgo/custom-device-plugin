package main

import (
	"flag"
	"time"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/knwgo/custom-device-plugin/pkg/server"
)

var (
	kubeletSocketPath, devicePluginPath string
	resourceName                        string
	devicePath                          string
)

func main() {
	flag.StringVar(&kubeletSocketPath, "kubelet-socket-path", pluginapi.KubeletSocket, "kubelet socket path")
	flag.StringVar(&devicePluginPath, "device-plugin-path", pluginapi.DevicePluginPath, "device plugin path")
	flag.StringVar(&resourceName, "resource-name", "example.com/foo", "resource name")
	flag.StringVar(&devicePath, "device-path", "/etc/custom-dev/", "resource path")
	klog.InitFlags(nil)

	flag.Parse()

	defer klog.Flush()
	klog.Infof("kubeletSocketPath: %s, devicePluginPath: %s, resourceName: %s, devicePath: %s", kubeletSocketPath, devicePluginPath, resourceName, devicePath)

	cdp := server.NewDPServer(kubeletSocketPath, devicePluginPath, resourceName, devicePath)
	if err := cdp.Run(); err != nil {
		klog.Fatal(err)
	}

	if err := cdp.RegisterToKubelet(); err != nil {
		klog.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Fatal(err)
	}
	defer func() {
		_ = watcher.Close()
	}()
	if err := watcher.Add(devicePluginPath); err != nil {
		klog.Fatal(err)
	}

	klog.Info("watching kubelet.sock")
	for {
		select {
		case event := <-watcher.Events:
			if event.Name == kubeletSocketPath && event.Op&fsnotify.Create == fsnotify.Create {
				klog.Warning("kubelet.sock recreated")
				time.Sleep(time.Second)
				klog.Fatalf("%s created, restarting", kubeletSocketPath)
			}
		case err := <-watcher.Errors:
			klog.Fatalf("watch kubelet sock error: %s", err)
		}
	}
}
