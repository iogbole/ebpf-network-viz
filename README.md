
# Monitoring TCP Retransmissions with eBPF, Go, and Prometheus: A Beginners guide to eBPF. 

Refer to this blog post for details on the background and motivation behind this project [https://www.israelo.io/blog/ebpf-net-viz/](https://www.israelo.io/blog/ebpf-net-viz/). 

## The Solution

The diagram below depicts the solution. 

<p align="center">
<img width="1510" alt="the solution" src="https://user-images.githubusercontent.com/2548160/273732796-16810c09-bf82-4bcb-a2ac-ca3ab04bfbb1.png">
</p>

## Preparation and Environment Setup
Before we begin, it's important to have your development environment properly configured. While this blog isn't an exhaustive tutorial, I'll outline the key prerequisites briefly. 

### **Using Lima on MacOS**
If you're a MacOS user like me, Lima is an excellent way to emulate a Linux environment. It's simple to set up and meshes seamlessly with your existing workflow. To kick things off with Lima, follow these steps:

1. Install Lima and launch it with the [ebpf-vm.yaml](https://github.com/iogbole/ebpf-network-viz/blob/main/ebpf-vm.yaml) file:

    ```bash
    limactl start ebpf-vm.yaml
    limactl shell ebpf-vm
    ```
2. If you're fond of Visual Studio Code, you can connect to the Lima VM via SSH:

    ```bash
    limactl show-ssh ebpf-vm
    ```

    Subsequently, use the SSH command to link up with the remote server through Visual Studio Code.

3. After establishing the connection, clone the required repository:

    ```bash
    git clone https://github.com/iogbole/ebpf-network-viz.git
    ```

### **Manual Setup on Linux**

If you’re opting for a manual setup on Linux, refer to the script section in the [ebpf-vm.yaml](https://github.com/iogbole/ebpf-network-viz/blob/main/ebpf-vm.yaml#L18) file.

With your environment now primed, you’re all set to delve into the fascinating world of eBPF!


## Run 

1. `Make`  : To compile the eBPF code and run main.go 
2. `./run_prom.sh` : To start Prometheus
3. `./create_tcp_chaos.sh` : To start `tc` and generate TCP requests. 


## Observe 

Head over to your Prometheus interface and type `tcp_retransmissions_total` into the query bar. Switch to the graph view and marvel at the results of your hard work.

You're now in a position to set up alerts for TCP retransmissions. A common benchmark to consider is that a retransmission rate of 2% or greater generally indicates network issues that warrant attention.

So grab a cup of coffee, sit back, and enjoy the fruit of your labour!


<p align="center">
<img width="1510"  src="https://user-images.githubusercontent.com/2548160/273732219-e4b7bcf0-5d4a-456a-8197-543ecbcea061.png">
</p>
