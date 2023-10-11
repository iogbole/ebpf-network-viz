
# Monitoring TCP Retransmissions with eBPF, Go, and Prometheus: A Beginners guide to eBPF. 

Refer to this blog post for details - https://www.israelo.io/blog/ebpf-net-viz/ 

## The Ghost in the Network: TCP Retransmissions

TCP retransmissions aren't inherently bad; they're a fundamental part of how TCP/IP networks function. However, when they occur frequently, they can signify network issues that lead to poor application performance. A high number of retransmissions can cause:

* **Increased Latency**: Packets have to be sent again, which takes extra time.
* **Higher CPU Usage**: Both sending and receiving systems have to do additional work to handle the retransmissions.
* **Bandwidth Inefficiency**: Retransmissions consume bandwidth that could be better used by new data.
* **User Experience Degradation**: All the above contribute to a laggy or suboptimal user experience.

<p align="center">
<img width="600" alt="tcp retransmission" src="https://github-production-user-asset-6210df.s3.amazonaws.com/2548160/273732239-ec8dd025-ea85-4e7f-9ef3-0063ff75f1e0.png">
</p>

Imagine working on a high-speed, low-latency product and encountering intermittent slowdowns in data transmission. This situation can be tricky to diagnose and could bring your product to its knees. When I faced this issue, I took it upon myself to delve deep and understand what was happening under the hood. **Wireshark led me to the root cause: excessive TCP retransmissions due to a faulty firewall policy**.

One can easily trigger TCP retransmission, by executing: 

```bash
sudo tc qdisc add dev eth0 root netem loss 10% delay 100ms
```
and it will surely mess up your network performance and introduce high CPU usage. I was once crazy enough to use 50% on an EC2 instance and it booted me out of SSH connection until I restarted the node via the console.  **Do not try this out at home ;)** 


## Why eBPF? 
eBPF is a revolutionary technology that allows users to extend the functionality of the Linux kernel without having to modify the kernel code itself. It is essentially a lightweight, sandboxed virtual machine that resides within the kernel, offering secure and verified access to kernel memory.

Moreso, eBPF code is typically written in a restricted subset of the C language and compiled into eBPF bytecode using a compiler like Clang/LLVM. This bytecode undergoes rigorous verification to ensure that it cannot intentionally or inadvertently jeopardize the integrity of the kernel. Additionally, eBPF programs are guaranteed to execute within a finite number of instructions, making them suitable for performance-sensitive use cases like observability and network security.

Here are some of the key benefits of using eBPF:

* **Safety and security**: eBPF programs are sandboxed and verified, which means that they cannot harm the kernel or the system as a whole.
* **Performance**: eBPF programs are extremely efficient and can be used to implement complex functionality without impacting system performance.
* **Flexibility**: eBPF can be used to implement a wide range of functionality, including network monitoring, asset discovery, security, profiling, performance tracing, and more.

Functionally, eBPF allows you to run this restricted C code in response to various events, such as timers, network events, or function calls within both the kernel and user space. These event hooks are often referred to as 'probes'—`kprobes` for kernel function calls, `uprobes` for user-space function calls, and `tracepoints` for pre-defined hooks in the Linux kernel.

In the context of this blog post, we'll be focusing on `tracepoints`, specifically leveraging the <code><em>tcp_retransmit_skb</em></code>  tracepoint for monitoring TCP retransmissions.  

If you are completely new to eBPF, I recommend checking out the resources in the reference section below, starting with [What is eBPF](https://ebpf.io/what-is-ebpf/)?

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

## The Solution

The diagram below depicts the solution. 

<p align="center">
<img width="1510" alt="the solution" src="https://user-images.githubusercontent.com/2548160/273732796-16810c09-bf82-4bcb-a2ac-ca3ab04bfbb1.png">
</p>

### Overview of the Components

This is how the code works at a very high level:

1. **Bytecode Loaded by Go**: The eBPF bytecode is loaded into the kernel using a Go program, which makes use of the `github.com/cilium/ebpf` package.

2. **eBPF Code Hooks to Tracepoints**: The eBPF program uses the `tracepoint/tcp/tcp_retransmit_skb` to monitor TCP retransmissions. This allows the code to trigger whenever a TCP packet is retransmitted.

2. **Collect Retransmission Events**: The data relating to the retransmitted packets—such as IP addresses, ports, and the protocol family are collected in a structured manner.

4. **Use of eBPF Maps**: eBPF maps are used to communicate between the eBPF code running in the kernel and the Go application running in user space.

5. **Perf Buffer**: A perf event buffer is used to read the events generated by the eBPF code.

6. **Exposed to HTTP**: The Go application exposes the metrics over HTTP on port 2112.

7. **Prometheus Scrapes Metrics**: Finally, Prometheus is configured to scrape these exposed metrics for monitoring or alerting purposes.
