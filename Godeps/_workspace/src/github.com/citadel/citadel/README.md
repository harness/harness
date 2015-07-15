## Citadel

A docker centric package for scheduling containers on a docker cluster. 


### Workflow

1. User provides a container type to the scheduler specifying the image, cpus, memory, and other 
constraints required in the decision making of where this container should be run on the cluster.
This is the json representation of what a sample request would look like:

  ```json
  {
      "image": "crosbymichael/redis",
      "cpus": "0.2",
      "memory": 512,
      "type": "service"
  }
  ```

2. Depending on the container type, service, batch, etc..., a specific scheduler is passed the container 
information for making the scheduling decision.  After the scheduler has made the decision it will send
it's decison on where this container should run to the cluster manager.  Schedulers can run in parallel
to make decisions but the cluster manager keeps the most up to date state of the cluster and gives the
scheduler and yes/no if the container can run on that resource.  

3. After the cluster manager approves the change in state of the cluster then decision is returned back
to the original requester along with information about the resource that can run the container.  They
are free to act on that information as they please.  A sample response json representation of a response
will look like:

  ```json
  {
      "container":{
          "image": "crosbymichael/redis",
          "cpus": "0.2",
          "memory": 512,
          "type": "service"
      },
      "resource": {
          "id": "12345abc",
          "addr": "192.168.56.8:4243",
          "total_cpus": 8,
          "total_memory": 32000
     }
  }
  ```

4. The cluster manager will need a way to receive and modify the cluster state.  This will be provided
by a `Registry` that the consumer of the library provides.


### Concepts

#### Cluster Manager

The cluster manager is responsible to for making changes to the state of the cluster.

#### Registry

The registry stores the state of the cluster and can be queried and modified by the cluster manager.
Th registry is not implemented by citadel

#### Scheduler

A cluster can have more than one scheduler.  Schedulers can be used together or alone.  Decisions 
by the schedulers can be made in parallel with disputes about resource placement handled when the 
scheduler wishes to modify the cluster state with the cluster manager.

The scheduler chosen for a specific container is dependent on the `type` of container being run.
A resource manager maybe needed because the schedulers can make decisions about eligble resources
where the container should run but the resource manager needs to have the say if that resource
is able to run the container and how to best utilize resources in the cluster.

#### Design questions

* Should disk and volumes be a resource like memory and cpu?
* Are ports a resource like memory and cpu or is this not our problem? 
* Should the runs be abstracted into the package so that the cluster manager knows of a failure?
* Should we only return one resource of where the container should run or should we return all
available resources? 


#### Wish list

* Monitor live resource usage instead of basing information off of a static reservation.
* Monitor live container metrics in the cluster to learn how specific images are using
resources and if they match the resources requested.
