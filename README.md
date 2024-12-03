[![Go Report Card](https://goreportcard.com/badge/github.com/knwgo/custom-device-plugin)](https://goreportcard.com/report/github.com/knwgo/custom-device-plugin)

an example K8s Device Plugin that can customize resource names

### Basic usage
**deploy custom device plugin**

`kubectl apply -f deploy/daemonset.yaml`

**generate device in node**

```shell
# in node
touch /etc/custom-dev/dev0
```

**check custom resource**

```shell
kubectl describe node <you node>
# You should see output similar to the following
Capacity:
  cpu:                8
  ephemeral-storage:  91957160Ki
  example.com/foo:    1
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  hugepages-32Mi:     0
  hugepages-64Ki:     0
  memory:             8027168Ki
  pods:               110
Allocatable:
  cpu:                8
  ephemeral-storage:  91957160Ki
  example.com/foo:    1
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  hugepages-32Mi:     0
  hugepages-64Ki:     0
  memory:             8027168Ki
  pods:               110
```
`example.com/foo` is the default resource name

### Advanced Usage

#### Custom resource name
specify the startup parameter `--resource-name` to customize the resource name

example: `--resource-name nvidia.com/gpu`


#### Device status
devices can customize `Numa Node` and `Health` status, just define the `Json` content in the device file 

example dev file content:
```json
{
  "Nodes": [0,1],
  "Unhealthy": false
}
```