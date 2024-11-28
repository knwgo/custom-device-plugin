package main

import (
	"flag"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/knwgo/custom-device-plugin/pkg/server"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
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
			klog.Infof("watch kubelet events: %s, event name: %s, isCreate: %v", event.Op.String(), event.Name, event.Op&fsnotify.Create == fsnotify.Create)
			if event.Name == kubeletSocketPath && event.Op&fsnotify.Create == fsnotify.Create {
				time.Sleep(time.Second)
				log.Fatalf("inotify: %s created, restarting", kubeletSocketPath)
			}
		case err := <-watcher.Errors:
			log.Fatalf("inotify: %s", err)
		}
	}
}