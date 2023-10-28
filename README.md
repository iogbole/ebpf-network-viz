
#  Getting Started with eBPF: Monitoring TCP Retransmissions with eBPF, Go and Prometheus 

> [!IMPORTANT]  
> Refer to this blog post for details on the background and motivation behind this experiment -  **[https://www.israelo.io/blog/ebpf-net-viz/](https://www.israelo.io/blog/ebpf-net-viz/)**



## Usage 

1. `Make` : To compile the eBPF code and run main.go 
2. `./run_prom.sh` : To start Prometheus
3. `./create_tcp_chaos.sh` : To start `tc` and generate TCP requests. 


## How it works 

The diagram below depicts the solution.  

Read the blog post for details [https://www.israelo.io/blog/ebpf-net-viz/](https://www.israelo.io/blog/ebpf-net-viz/)



<p align="center">
<img width="1510" alt="the solution" 
src="https://user-images.githubusercontent.com/2548160/274510771-99bb4583-c7be-4e3e-83fc-283ea99d0195.png">
</p>



## Observe 

Head over to your Prometheus interface and type `tcp_retransmissions_total` into the query bar. Switch to the graph view and marvel at the results of your hard work.

<p align="center">
<img width="1510"  src="https://user-images.githubusercontent.com/2548160/274725653-9b2ac550-01cc-4015-befb-9539a9b38d03.gif">
</p>


### **Using Lima on MacOS**
If you're a MacOS user like me, Lima is an excellent way to emulate a Linux environment. To kick things off with Lima, follow these steps:

1. Install Lima and launch it with the [ebpf-vm.yaml](https://github.com/iogbole/ebpf-network-viz/blob/main/ebpf-vm.yaml) file:

    ```bash
    limactl start ebpf-vm.yaml
    limactl shell ebpf-vm
    ```
2. If use use Visual Studio Code, you can connect to the Lima VM via SSH:

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


