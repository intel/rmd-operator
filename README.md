# Intel RMD Operator
[![Go Report Card](https://goreportcard.com/badge/github.com/intel/rmd-operator)](https://goreportcard.com/report/github.com/intel/rmd-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

----------
Kubernetes Operator designed to provision and manage Intel [Resource Management Daemon (RMD)](https://github.com/intel/rmd) instances in a Kubernetes cluster.
----------

Table of Contents
=================

   * [Intel RMD Operator](#intel-rmd-operator)
      * [Notice: Changes Introduced for RMD Operator v0.3](#notice-changes-introduced-for-rmd-operator-v03)
      * [Prerequisites](#prerequisites)
      * [Setup](#setup)
      * [Custom Resource Definitions (CRDs)](#custom-resource-definitions-crds)
         * [RmdConfig](#rmdconfig)
         * [RmdWorkload](#rmdworkload)
         * [RmdNodeState](#rmdnodestate)
      * [Static Configuration Aligned With the CPU Manager](#static-configuration-aligned-with-the-cpu-manager)
      * [Dynamic Configuration With the CPU Manager (Experimental)](#dynamic-configuration-with-the-cpu-manager-experimental)

## Notice: Changes Introduced for RMD Operator v0.3
* RmdConfig CRD: This object is introduced in **v0.3**. See explanation [below](#rmdconfig).
* RmdWorkload CRD: Additional spec fields `nodeSelector` see [example](#cache-nodeselector), `allCores` see [example](#cache-all-cores), `reservedCoreIds` see [example](#create-rmdworkload-1).
* RMD Node Agent is not deployed by default. See explanation [below](#rmdconfig).
* Pods requesting RDT features are no longer deleted by the operator. In **v0.2** and earlier, should a workload fail to post to RMD after being requested via a pod spec, the pod would be deleted and the RmdWorkload object was garbage collected as a result. This is no longer the case in **v0.3**. Instead, the pod is not deleted, but the reason for failure is displayed in the pod's child RmdWorkload status. The onus is on the user to verify that the workload was configured succesfully by RMD.
* Extended resources `intel.com/l3_cache_ways` represent the number of cache ways available in the [RMD guaranteed pool](https://github.com/intel/rmd#cache-poolsgroups). Cache ways are represented as devices and as such, a lightweight device plugin is deployed along with RMD for their discovery and advertisement. If running behind a proxy server, proxy settings must be configured in the plugin's [Dockerfile](https://github.com/intel/rmd-operator/blob/master/build/deviceplugin.Dockerfile#L3).

## Prerequisites
* Node Feature Discovery ([NFD](https://github.com/kubernetes-sigs/node-feature-discovery)) should be deployed in the cluster before running the operator. Once NFD has applied labels to nodes with capabilities compatible with RMD, such as *Intel L3 Cache Allocation Technology*, the operator can deploy RMD on those nodes. 
Note: NFD is recommended, but not essential. Node labels can also be applied manually. See the [NFD repo](https://github.com/kubernetes-sigs/node-feature-discovery#feature-labels) for a full list of features labels.
* A working RMD container image from the [RMD repo](https://github.com/intel/rmd) compatible with the RMD Operator (see compatiblilty table below).  

### Compatibility
|  RMD Version | RMD Operator Version |
| ------ | ------ |
| v0.1 | N/A |
| v0.2 | v0.1 |
| v0.3 | v0.2 |
| v0.3 | v0.3 |

## Setup

### Debug Mode
To use the operator with RMD in [debug mode](https://github.com/intel/rmd/blob/master/docs/UserGuide.md#run-the-service), the [port number](https://github.com/intel/rmd-operator/-/blob/master/build/manifests/rmd-ds.yaml#L20) of **build/manifests/rmd-ds.yaml** must be set to `8081` before building the operator. Debug mode is advised for testing only. 

### TLS Enablement
To use the operator with [RMD with TLS enabled](https://github.com/intel/rmd/blob/master/docs/UserGuide.md#access-using-https-over-tcp-connection-secured-by-tls), the [port number](https://github.com/intel/rmd-operator/blob/master/build/manifests/rmd-ds.yaml#L20) of **build/manifests/rmd-ds.yaml** must be set to `8443` before building the operator. Sample certificates are provided by the [RMD repository](https://github.com/intel/rmd/tree/master/etc/rmd/cert/client) and should be used for testing only. The user can generate their own certs for production and replace with those existing. The client certs for the RMD operator should be stored in the following locations in this repo before building the operator:

CA: **build/certs/public/ca.pem**

Public Key: **build/certs/public/cert.pem**

Private Key: **build/certs/private/key.pem**

### Build
*Note:* The operator deploys pods with the RMD container. The [Dockerfile](https://github.com/intel/rmd/blob/master/Dockerfile) for this container is located on the [RMD repo](https://github.com/intel/rmd) and is out of scope for this project. 

*Note:* If running behind a proxy server, proxy settings must be configured in the RMD Operator [Dockerfile](https://github.com/intel/rmd-operator/blob/master/build/Dockerfile#L3). 

The RMD image name is specified in the `RmdConfig` file located at **deploy/rmdconfig.yaml**. Alterations to the image name/tag should be made here.

Build go binaries for the operator and the node agent:

`make build`

Build docker images for the operator and the node agent:

`make images`

*Note:* The Docker images built are `intel-rmd-operator:latest` and `intel-rmd-node-agent:latest`. Once built, these images should be stored in a remote docker repository for use throughout the cluster.

### Deploy
The **deploy** directory contains all specifications for the required RBAC objects. These objects can be inspected and deployed individually or created all at once using rbac.yaml:

`kubectl apply -f deploy/rbac.yaml`

Create RmdNodeState CRD:

`kubectl apply -f deploy/crds/intel.com_rmdnodestates_crd.yaml`

Create RmdWorkloads CRD:

`kubectl apply -f deploy/crds/intel.com_rmdworkloads_crd.yaml`

Create RmdConfigs CRD:

`kubectl apply -f deploy/crds/intel.com_rmdconfigs_crd.yaml`

Create Operator Deployment:

`kubectl apply -f deploy/operator.yaml`

All of the above `kubectl` commands can be done by:

`make deploy`

Note: For the operator to deploy and run RMD instances, an up to date RMD docker image is required.

### Quickstart

All above commands for build, images, deploy can be done by:

`make all`

## Custom Resource Definitions (CRDs)

### RmdConfig
The RmdConfig custom resource is the object that governs the overall deployment of RMD instances across a cluster. 
The RmdConfig spec consists of:
-   `rmdImage`: This is the name/tag given to the RMD container image that will be deployed in a DaemonSet by the operator.
-   `rmdNodeSelector`: This is a key/value map used for defining a list of node labels that a node must satisfy in order for RMD to be deployed on it. If no `rmdNodeSelector` is defined, the default value is set to the single feature label for RDT L3 CAT (`"feature.node.kubernetes.io/cpu-rdt.RDTL3CA": "true"`).
-   `deployNodeAgent`: This is a boolean flag that tells the operator whether or not to deploy the node agent along with the RMD pod. The node agent is only necessary for requesting RDT features via the pod spec. This approach is experimental and as such, is disabled by default.

The RmdConfig status represents the nodes which match the `rmdNodeSelector` and have RMD deployed.

#### Example
See `deploy/rmdconfig.yaml` 
````yaml
apiVersion: intel.com/v1alpha1
kind: RmdConfig
metadata:
    name: rmdconfig
spec:
    rmdImage: "rmd:latest"
    rmdNodeSelector:
        "feature.node.kubernetes.io/cpu-rdt.RDTL3CA": "true"
    deployNodeAgent: false    
````
**Note:** Only one RmdConfig object is necessary per cluster. This is enforced by virtue of the default naming convention `"rmdconfig"`.

### RmdWorkload
The RmdWorkload custom resource is the object used to define a workload for RMD.
RmdWorkload objects can be created **directly** via the RmdWorkload spec.

**Direct configuration is the recommended approach** as it affords the user more control over specific cores and specific nodes on which they wish to configure a particular RmdWorkload. The Kubelet's CPU Manager can then allocate CPUs on pre-configured nodes to containers, as described in more detail [here](#recommended-approach-for-use-with-the-cpu-manager). This section describes how to create an RmdWorkload spec directly.

It is also possible to create RmdWorkloads **automatically** via the pod spec. Automatic configuration utilizes pod annotations and the `intel.com/l3_cache_ways` extended resource to create an RmdWorkload for the same CPUs that are allocated to the pod.
**Automatic configuration has a number of limitations and is less stable than direct configuration.** The automatic configuration approach is described [later](#experimental-approach-for-use-with-the-cpu-manager) in this document.


#### Examples
See `samples` directory for RmdWorkload templates.

##### Cache (Specific Core IDs)
See `samples/rmdworkload-guaranteed-cache.yaml`
````yaml
apiVersion: intel.com/v1alpha1
kind: RmdWorkload
metadata:
    name: rmdworkload-guaranteed-cache
spec:
    coreIds: ["0-3","6","8"]
    rdt:
        cache:
            max: 2
            min: 2
    nodes: ["worker-node-1", "worker-node-2"]
````
This workload requests cache from the guaranteed group for CPUs 0 to 3, 6 and 8 on nodes "worker-node-1" and "worker-node-2". See [intel/rmd](https://github.com/intel/rmd#cache-poolsgroups) for details on cache pools/groups.

**Note**: Replace "worker-node-1" and "worker-node-2" in *nodes* field with the actual node name(s) you wish to target with your RmdWorkload spec. 

Creating this workload is the equivalent of running the following command for each node: 
````
$ curl -H "Content-Type: application/json" --request POST --data \
        '{"core_ids":["0","1","2","3","6","8"],
            "rdt" : { 
                "cache" : {"max": 2, "min": 2 }
            }    
        }' \
        https://hostname:port/v1/workloads
````
##### Cache (All Cores)
See `samples/rmdworkload-guaranteed-cache-all-cores.yaml`
````yaml
apiVersion: intel.com/v1alpha1
kind: RmdWorkload
metadata:
    name: rmdworkload-guaranteed-cache
spec:
    allCores: true
    rdt:
        cache:
            max: 2
            min: 2
    nodes: ["worker-node-1", "worker-node-2"]
````
This workload requests cache from the guaranteed group for **all** CPUs on nodes "worker-node-1" and "worker-node-2". See [intel/rmd](https://github.com/intel/rmd#cache-poolsgroups) for details on cache pools/groups.

**Note**: If `allCores` is `true` and a `coreIds` list is also specified, `allCores` will take precedence and the specified `coreIds` list will be redundant.

**Note**: Replace "worker-node-1" and "worker-node-2" in *nodes* field with the actual node name(s) you wish to target with your RmdWorkload spec. 

Creating this workload is the equivalent of running the following command for each node: 
````
$ curl -H "Content-Type: application/json" --request POST --data \
        '{"core_ids":["0","1","2","3","6","8" . . . .],
            "rdt" : { 
                "cache" : {"max": 2, "min": 2 }
            }    
        }' \
        https://hostname:port/v1/workloads
````
##### Cache (NodeSelector)
See `samples/rmdworkload-guaranteed-cache-nodeselector.yaml`
````yaml
apiVersion: intel.com/v1alpha1
kind: RmdWorkload
metadata:
    name: rmdworkload-guaranteed-cache
spec:
    allCores: true
    rdt:
        cache:
            max: 2
            min: 2
    nodeSelector: 
      feature.node.kubernetes.io/cpu-rdt.RDTL3CA: "true"
````
This workload requests cache from the guaranteed group for **all** CPUs on all nodes with feature label `feature.node.kubernetes.io/cpu-rdt.RDTL3CA=true`. See [intel/rmd](https://github.com/intel/rmd#cache-poolsgroups) for details on cache pools/groups.

The `nodeSelector` label is useful for cluster partitioning. For example, a number of nodes can be grouped together by a common label and pre-provisioned with particular RDT features/settings via a single RMD workload. This node group can then be targeted by workloads that require such settings via existing K8s constructs such as [`nodeAffinity`](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes-using-node-affinity/). Please see [recommended approach for for use with the CPU Manager](#recommended-approach-for-use-with-the-cpu-manager) for a more detailed example.

**Note**: If `nodeSelector` is specified and a `nodes` list is also specified, `nodeSelector` will take precedence and the specified `nodes` list will be redundant.

Creating this workload is the equivalent of running the following command for each node: 
````
$ curl -H "Content-Type: application/json" --request POST --data \
        '{"core_ids":["0","1","2","3","6","8" . . . .],
            "rdt" : { 
                "cache" : {"max": 2, "min": 2 }
            }    
        }' \
        https://hostname:port/v1/workloads
````
##### Memory Bandwidth Allocation (MBA)

**Note:** MBA can only be requested **with** guaranteed cache. See [intel/rmd](https://github.com/intel/rmd) for more information on MBA.

See `samples/rmdworkload-guaranteed-cache-mba.yaml`
````yaml
apiVersion: intel.com/v1alpha1
kind: RmdWorkload
metadata:
    name: rmdworkload-guaranteed-cache-mba
spec:
    coreIds: ["0-3"]
    rdt:
        cache:
            max: 2
            min: 2
        mba:
            percentage: 50
    nodes: ["worker-node-1", "worker-node-2"]
````
This workload requests cache from the guaranteed group for CPUs 0 to 3 on nodes "worker-node-1" and "worker-node-2" while also assigning 50% MBA to those CPUs. See [intel/rmd](https://github.com/intel/rmd#cache-poolsgroups) for details on cache pools/groups.

**Note**: Replace "worker-node-1" and "worker-node-2" in *nodes* field with the actual node name(s) you wish to target with your RmdWorkload spec. 

Creating this workload is the equivalent of running the follwing command for each node: 
````
$ curl -H "Content-Type: application/json" --request POST --data \
        '{"core_ids":["0","1","2","3"],
            "rdt" : { 
                "cache" : {"max": 2, "min": 2 }
                "mba" : {"percentage": 50 }
            }    
        }' \
        https://hostname:port/v1/workloads
````
##### P-State

**Note:** P-State settings are only configurable if the [RMD P-State plugin has been loaded](https://github.com/intel/rmd/blob/master/docs/ConfigurationGuide.md#pstate-section).  

See `samples/rmd-workload-guaranteed-cache-pstate.yaml`
````yaml
apiVersion: intel.com/v1alpha1
kind: RmdWorkload
metadata:
    name: rmdworkload-guaranteed-cache-pstate
spec:
    coreIds: ["4-7"]
    rdt:
        cache:
            max: 2
            min: 2
    plugins:        
        pstate:
            ratio: "3.0"
            monitoring: "on"
    nodes: ["worker-node-1", "worker-node-2"]
````
This workload expands on the previous example with manually specified parameters with P-State plugin enabled. 

Creating this workload is the equivalent of running the following command for each node: 
````
$ curl -H "Content-Type: application/json" --request POST --data \
        '{"core_ids":["4","5","6","7"],
            "rdt": {         
                "cache" : {"max": 2, "min": 2 }
            }
            "plugins" : {
                "pstate" : {"ratio": 3.0, "monitoring" : "on"}
            }
       }' \
       https://hostname:port/v1/workloads
````

##### Create RmdWorkload

`kubectl create -f samples/rmdworkload-guaranteed-cache.yaml`


##### List RmdWorkloads
`kubectl get rmdworkloads`


##### Display a particular RmdWorkload:
`kubectl describe rmdworkload rmd-workload-guaranteed-cache`

````
Name:         rmdworkload-guaranteed-cache
Namespace:    default
API Version:  intel.com/v1alpha1
Kind:         RmdWorkload
Spec:
  Cache:
    Max:  2
    Min:  2
  Core Ids:
    0-3
    6
    8
  Nodes:
    worker-node-1
    worker-node-2
Status:
  Workload States:
    worker-node-1:
      Core Ids:
        0-3
        6
        8
      Cos Name:  0-3_6_8-guarantee
      Id:        1
      Rdt:
        Cache:
          Max:  2
          Min:  2
      Response:  Success: 200
      Status:    Successful
    worker-node-2:
      Core Ids:
        0-3
        6
        8
      Cos Name:  0-3_6_8-guarantee
      Id:        1
      Rdt:
        Cache:
          Max:  2
          Min:  2
      Response:  Success: 200
      Status:    Successful
````
This displays the RmdWorkload object including the spec as defined above and the status of the workload. Here, the status shows that this workload was configured successfully on nodes "worker-node-1" and "worker-node-2".

##### Delete RmdWorkload
When the user deletes an RmdWorkload object, a delete request is sent to the RMD API on every RMD instance on which that RmdWorkload is configured.

`kubectl delete rmdworkload rmdworkload-guaranteed-cache`

Note: If the user only wishes to delete the RmdWorkload from a specific node, that node should be removed from the RmdWorkload spec's "nodes" field and then apply the RmdWorkload object.

`kubectl apply -f samples/rmdworkload-guaranteed-cache.yaml`

### RmdNodeState
The RmdNodeState custom resource is created for each node in the cluster which has RMD running. The purpose of this object is to allow the user to view all running workloads on a particular node at any given time.
Each RmdNodeState object will be named according to its corresponding node (ie `rmd-node-state-<node-name>`).


##### List all RmdNodeStates on the cluster
`kubectl get rmdnodestates`

##### Display a particular RmdNodeState such as the example above
`kubectl describe rmdnodestate rmd-node-state-worker-node-1`

````
Name:         rmd-node-state-worker-node-1
Namespace:    default
API Version:  intel.com/v1alpha1
Kind:         RmdNodeState
Spec:
  Node:      worker-node-1
  Node UID:  75d03574-6991-4292-8f16-af43a8bfa9a6
Status:
  Workloads:
    rmdworkload-guaranteed-cache:
      Cache Max:  2
      Cache Min:  2
      Core IDs:   0-3,6,8
      Cos Name:   0-3_6_8-guarantee
      ID:         1
      Origin:     REST
      Status:     Successful
    rmdworkload-guaranteed-cache-pstate:
      Cache Max:  2
      Cache Min:  2
      Core IDs:   4-7
      Cos Name:   4-7-guarantee
      ID:         2
      Origin:     REST
      Status:     Successful
````
This example displays the RmdNodeState for worker-node-1. It shows that this node currently has two RMD workloads configured successfully.

## Static Configuration Aligned With the [CPU Manager](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/)
This approach is reliable, but has drawbacks such as potentially under utilised resources. As such, it may be more suited to nodes with lesser CPU resources (eg VMs). 

In order to have total confidence in how the CPU Manager will allocate CPUs to containers, it is necessary to pre-provision all *Allocatable* (i.e. *shared pool* - *reserved-cpus*) CPUs on a specific node (or group of nodes) with a common configuration. These nodes are then used for containers with CPU requirements that match the pre-provisioned configuration. 

Should there be a need to [designate these nodes](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/#example-use-cases) exclusively to containers that require these pre-provisioned CPUs, these nodes can also be tainted. As a result, only suitable pods tolerating the taint will be scheduled to these nodes.

The following is an example of how this is achieved. Please read all necessary documentation linked in the example before attempting this approach or similar.

### Example
#### Node Setup
* Set the Kubelet flag `reserved-cpus` with a list of specific [CPUs to be reserved for system and Kubernetes daemons](https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/#explicitly-reserved-cpu-list):
  
  `--reserved-cpus=0-3`
  
  Reserved CPUs 0-3 are no longer **exclusively** allocatable by the CPU Manager. Simply put, a container requesting exclusive CPUs cannot be allocated CPUs 0-3.
  
  CPUs 0-3 do, however, remain in the CPU Manager's shared pool.
  
* Apply a label to this node representing the configuration you wish to apply (eg `node.guaranteed.cache.only=true`). Please read K8s documentation on [node labelling](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes/).
    
  `kubectl label node <node-name> node.guaranteed.cache.only=true`
    
* **Optional**: Apply a taint to this node in order to only allow pods that tolerate the taint to be scheduled to this node. Please read K8s documentation on [node tainting](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/).
    
  `kubectl taint nodes <node-name> node.guaranteed.cache.only=true:NoSchedule`
    
* Repeat these steps for all nodes that are to be designated, ensuring the same `reserved-cpus` list is used for all nodes.

#### Create RmdWorkload

This workload is to be configured for all *Allocatable* CPUs on the desired node(s). This is done by setting `allCores` to true and specifying `reservedCoreIds` with the **same CPU list that has been reserved by the Kubelet** in the earlier step. 

This workload will be configured on all nodes that have been labelled `node.guaranteed.cache.only=true` in the earlier step. This is achieved through the RmdWorkload spec field `nodeSelector`.

**Note:** The list of `reservedCoreIds` is only taken into consideration when `allCores` is set to true. 

See `samples/rmdworkload-guaranteed-cache-allocatable.yaml`
````yaml
apiVersion: intel.com/v1alpha1
kind: RmdWorkload
metadata:
    name: rmdworkload-guaranteed-cache
spec:
    allCores: true
    reservedCoreIds: ["0-3"]
    rdt:
        cache:
            max: 2
            min: 2
    nodeSelector: 
      node.guaranteed.cache.only: "true"
      
````
#### Create Pod

This pod is provided with a `nodeSelector` for the designated node label, and a `taintToleration` for the designated node taint. 

This is a guaranteed pod requesting exclusive CPUs.

Please read K8s documentation on [assigning pods to nodes](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#:~:text=Run%20kubectl%20get%20nodes%20to,the%20node%20you've%20chosen).


 ````
 apiVersion corev1
kind Pod
metadata:
  name: guaranteed-cache-pod
spec:
  nodeSelector:
    node.guaranteed.cache.only: "true"
  tolerations:
  - key: "node.guaranteed.cache.only"
    operator: Equal
    value: "true"
    effect: NoSchedule
  containers:
  - name: container1
    resources:
      requests:
        memory: "64Mi"
        cpu: 3
      limits:
        memory: "64Mi"
        cpu: 3
````
#### Result

The pod will be scheduled to a designated `node.guaranteed.cache.only` node and the pod's container will be allocated 3 CPUs that have been pre-configured by the `rmdworkload-guaranteed-cache` RmdWorkload.
 
## Dynamic Configuration With the [CPU Manager](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/) (Experimental)
It is also possible for the operator to create an RmdWorkload **automatically** by interpreting resource requests and [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) in the pod spec.

### Warning: This approach is experimental and is not recommended in production. Intimate knowledge of the workings of the CPU Manager and existing cache resources are required. 

Under this approach, the user creates a pod with a container requesting **exclusive** CPUs from the Kubelet CPU Manager and available cache ways from the guaranteed pool. If additional capabilities such as MBA are desired, the pod must also contain RMD specific pod annotations to describe the desired RmdWorkload.
It is then the responsiblity of the operator and the node agent to do the following:
*  Extract the RMD related data passed to the pod spec by the user.
*  Discover which CPUs have been allocated to the container by the CPU Manager.
*  Create the RmdWorkload object based on this information.

The following criteria must be met in order for the operator to succesfully create an RmdWorkload with guaranteed cache for a container based on the pod spec.
*  The RmdConfig `deployNodeAgent` field must be set to `true` and updated via `kubectl apply -f deploy/rmdconfig.yaml`.
*  [RMD cache pools] (https://github.com/intel/rmd#cache-poolsgroups) configured correctly (out of scope for this project).
*  The container must request extended resource `intel.com/l3_cache_ways`. 
*  The container must also [request exclusive CPUs from CPU Manager](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/#static-policy).
*  The Kubelet's [Topology Manager](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager) should be configured with the `single-numa-node` [policy](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/#topology-manager-policies) on the node which RMD is deployed. The reason for this is to reduce the possibility of a workload failing after container creation. This might happen if cache ways are not available from the same NUMA node as the allocated CPUs. Cache ways are advertised as devices with an associated NUMA node. This enables the Topology Manager to align cache ways and CPUs, helping to mitigate the risk of RMD workload failure. However, this eventuality is still possible and will result in a `Topology Affinity Error` should more cache ways be requested than can be satisfied on a single NUMA node.

The following *additional* criteria must be met in order for the operator to succesfully create an RmdWorkload with with guaranteed cache *and* additional features such as MBA for a container based on the pod spec.
*  Pod annotations pertaining to the container requesting RMD features must be prefixed with that container's name. See example and [table](https://github.com/nolancon/rmd-operator/blob/v0.2/README.md#pod-annotaions-naming-convention) below.

### Example: Single Container
See `samples/pod-guaranteed-cache.yaml`

````yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-guaranteed-cache
spec:
  containers:
  - name: container1
    image: clearlinux/os-core:latest
    # keep container alive with sleep infinity
    command: [ "sleep" ]
    args: [ "infinity" ]
    resources:
      requests:
        memory: "64Mi"
        cpu: 3
        intel.com/l3_cache_ways: 2
      limits:
        memory: "64Mi"
        cpu: 3
        intel.com/l3_cache_ways: 2
````
This pod spec has one container requesting 3 exclusive CPUs and 2 cache ways. The number of cache ways requested is also interpreted as the value for `max cache` **and** `min cache` for the RmdWorkload. This means the container will be allocated 2 cache ways from [RMD's guaranteed pool](https://github.com/intel/rmd#cache-poolsgroups).

This pod will trigger the operator to automatically create an RmdWorkload for `container1` called `pod-guaranteed-cache-rmdworkload-container1`

#### Create Pod
`kubectl create -f sample/pod-guaranteed-cache.yaml`

#### Display RmdWorkload
If successful, the RmdWorkload will be created with the naming convention `<pod-name>-rmd-workload-<container-name>`

`kubectl describe rmdworkload pod-guaranteed-cache-rmdworkload-container1`

````
Name:         pod-guaranteed-cache-rmd-workload-container1
Namespace:    default
API Version:  intel.com/v1alpha1
Kind:         RmdWorkload
Spec:
  Core Ids:
    1
    2
    49
  Nodes:
    worker-node-1
  Plugins:
    Pstate:
  Rdt:
    Cache:
      Max:  2
      Min:  2
    Mba:
Status:
  Workload States:
    worker-node-1:
      Core Ids:
        1
        2
        49
      Cos Name:  1_2_49-guarantee
      Id:        22
      Plugins:
        Pstate:
      Rdt:
        Cache:
          Max:  2
          Min:  2
        Mba:
      Response:  Success: 200
      Status:    Successful
`````
This output displays the RmdWorkload which has been created succesfully based on the pod spec created above.
Note that CPUs 1,2 and 49 have been allocated to the container by the CPU Manager. As this RmdWorkload was created automatically via the pod spec, **the user has no control over which CPUs are used by the container**. 
In order to explicitly define which CPUs are to be allocated cache ways, the RmdWorkload must be created directly via the RmdWorkload spec and not the pod spec.

If **unsuccessful**, the RmdWorkload will be created with the naming convention `<pod-name>-rmd-workload-<container-name>` and the reason for failure of the workload to post to RMD will be displayed in the RmdWorkload status.

`kubectl describe rmdworkload pod-guaranteed-cache-rmdworkload-container1`

````
Name:         pod-guaranteed-cache-rmd-workload-container1
Namespace:    default
API Version:  intel.com/v1alpha1
Kind:         RmdWorkload
Spec:
  Core Ids:
    1
    2
    49
  Nodes:
    worker-node-1
  Plugins:
    Pstate:
  Rdt:
    Cache:
      Max:  2
      Min:  2
    Mba:
Status:
  Workload States:
    worker-node-1:
      Core IDs: <nil>
      Cos Name:  
      Id:        
       Plugins:
        Pstate:
      Rdt:
        Cache:
        Mba:
      Response:  Fail: Failed to validate workload. Reason: Workload validation in database failed. Details: CPU list [1] has been assigned
      Status:    
`````
This output displays the RmdWorkload which has been created based on the pod spec created above, however the corresponding workload has not been succesfully posted to RMD. The reason why this failure occurred is reflected in the `Response` field of the RmdWorkload status for `worker-node-1`.


### Example: Multiple Containers
See `samples/pod-multi-guaranteed-cache-mba.yaml`

````yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-multi-guaranteed-cache-mba
  annotations:
    container2_mba_percentage: "50"
spec:
  containers:
  - name: container1
    image: clearlinux/os-core:latest
    # keep container alive with sleep infinity
    command: [ "sleep" ]
    args: [ "infinity" ]
    resources:
      requests:
        memory: "64Mi"
        cpu: 2
        intel.com/l3_cache_ways: 2
      limits:
        memory: "64Mi"
        cpu: 2
        intel.com/l3_cache_ways: 2
  - name: container2
    image: clearlinux/os-core:latest
    # keep container alive with sleep infinity
    command: [ "sleep" ]
    args: [ "infinity" ]
    resources:
      requests:
        memory: "64Mi"
        cpu: 2
        intel.com/l3_cache_ways: 2
      limits:
        memory: "64Mi"
        cpu: 2
        intel.com/l3_cache_ways: 2
````
This pod spec has two container requesting 2 exclusive CPUs and 2 cache ways. The number of cache ways requested is interpreted as the value for `max cache` **and** `min cache` for the RmdWorkload. This means the container will be allocated 2 cache ways from [RMD's guaranteed pool](https://github.com/intel/rmd#cache-poolsgroups).
The `mba_percentage` value is specified for `container2` in the pod annotations.
The naming convention for RMD workload related annotations **must** follow the [table](https://github.com/nolancon/rmd-operator/blob/v0.2/README.md#pod-annotaions-naming-convention) below.

This pod will trigger the operator to automatically create **two** RmdWorkloads. One for `container1` called `pod-multi-guaranteed-cache-mba-rmdworkload-container1` and one for `container2` called `pod-multi-guaranteed-cache-mba-rmdworkload-container2` 
 
#### Create Pod
`kubectl create -f sample/pod-multi-guaranteed-cache-mba.yaml`

#### Display RmdWorkloads
If successful, the RmdWorkloads will be created with the naming convention `<pod-name>-rmd-workload-<container-name>`
 
`kubectl describe rmdworkload pod-guaranteed-cache-rmdworkload-container1`

````
Name:         pod-multi-guaranteed-cache-mba-rmd-workload-container1
Namespace:    default
API Version:  intel.com/v1alpha1
Kind:         RmdWorkload
Spec:
  Core Ids:
    1
    49
  Nodes:
    worker-node-1
  Plugins:
    Pstate:
  Rdt:
    Cache:
      Max:  2
      Min:  2
    Mba:
Status:
  Workload States:
    worker-node-1:
      Core Ids:
        1
        49
      Cos Name:  1_49-guarantee
      Id:        23
      Plugins:
        Pstate:
      Rdt:
        Cache:
          Max:  2
          Min:  2
        Mba:
      Response:  Success: 200
      Status:    Successful

`````
This output displays the RmdWorkload created succesfully for `container1` based on the pod spec created above.

`kubectl describe rmdworkload pod-guaranteed-cache-rmdworkload-container2`

````
Name:         pod-multi-guaranteed-cache-mba-9xhxt-rmd-workload-container2
Namespace:    default
API Version:  intel.com/v1alpha1
Kind:         RmdWorkload
Spec:
  Core Ids:
    2
    50
  Nodes:
    worker-node-1
  Plugins:
    Pstate:
  Rdt:
    Cache:
      Max:  2
      Min:  2
    Mba:
      Percentage:  50
Status:
  Workload States:
    worker-node-1:
      Core Ids:
        2
        50
      Cos Name:  2_50-guarantee
      Id:        24
      Plugins:
        Pstate:
      Rdt:
        Cache:
          Max:  2
          Min:  2
        Mba:
          Percentage:  50
      Response:  Success: 200
      Status:    Successful
````
This output displays the RmdWorkload created succesfully for `container2` based on the pod spec created above.

### Pod Annotaions Naming Convention
**Note**: Annotations **must** be prefixed with the relevant container name as shown below.
|  Specification | Container Name | Required Annotaion Name |
| ------ | ------ | ------ |
| Policy | container1 | container1_policy |
| P-State Ratio | container2 | container2_pstate_ratio |
| P-State Monitoring | test-container | test-container_pstate_monitoring |
| MBA Percentage | test-container-1 | test-container-1_mba_percentage |
| MBA Mbps | test-container2 | test-container2_mba_mbps |

Failure to follow the provided annotation naming convention will result in failure to create the desired workload. 

### Delete Pod and RmdWorkload
When an RmdWorkload is created by the operator based on a pod spec, that pod object becomes the owner of the RmdWorkload object it creates. Therefore when a pod that owns an RmdWorkload (or multiple RmdWorkloads) is deleted, all of its RmdWorkload children are automatically garbage collected and thus removed from RMD.

`kubectl delete pod rmd-workload-guaranteed-cache-pod-86676`

### Limitations in Creating RmdWorkloads via Pod Spec
*  Automatic configuration is only achievalbe with the native Kubernetes CPU Manager static policy.
*  The user has no control over which CPUs are configured with the automatically created RmdWorkload policy as the CPU Manager is in charge of CPU allocation.


Creating an RmdWorkload automatically via a pod spec is far less reliable than creating directly via an RmdWorkload spec. This is because the user no longer has the ability to explicitly define the specific CPUs on which the RmdWorkload will ultimately be configured.
CPU allocation for containers is the responsibility of the CPU Manager in Kubelet. As a result, the RmdWorkload will only be created **after** the pod is admitted. Once the RmdWorkload is created by the operator, the RmdWorkload information is sent to RMD in the form of an HTTPS post request. 
Should the post to RMD fail at this point, the reason for failure will be reflected in the RmdWorkload status. 

**Note:** It is important that the user always checks the RmdWorkload status after pod creation to validate that the workload has been configured correctly.

## Cleanup

To remove the RMD operator, related objects and workloads configured to RMD instances, complete the following steps.

### Delete RmdWorkloads

Delete RmdWorkload objects to allow the operator to perform workload removal from RMD instances:

`kubectl delete rmdworkloads --all`

**Note:** If an RmdWorkload has been created [automatically via the pod spec](#experimental-approach-for-use-with-the-cpu-manager), this RmdWorkload is a child of its corresponding pod. As a result, simply deleting this RmdWorkload will only result in the node agent reconciling the parent pod and the RmdWorkload will be recreated instantly. For this reason, pods that are parents of RmdWorkloads need to be deleted instead. This will result in the RmdWorkload being garbage collected by the operator and the workload will be removed from the relative RMD instance(s).

### Delete All Remaining Objects

`make remove`
