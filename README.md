# Monitoring TCP Retransmissions with eBPF, Go, and Prometheus: A Beginners guide to eBPF. 

As a Technical Product Manager, I do more than just manage products—I deeply engage with the technology that underpins them. In my case, this technology is eBPF (Extended Berkeley Packet Filter). Having recently completed my MBA, I picked up Liz Rice's ["Learning eBPF"](https://isovalent.com/books/learning-ebpf/) book. The book was so enlightening that I couldn't resist rolling up my sleeves to get hands-on with this revolunationary technology.

But why eBPF and, more specifically, why focus on monitoring TCP retransmissions? Well, a past nasty experience involving troubleshooting intermittent connectivity issues for an APM agent in production for a customer left me realising the need for better tools; Wireshark has its limitations. Had eBPF been in my toolkit back then, that daunting issue would have been far easier to diagnose and resolve.

This blog is intended to chronicle my hands-on exploration of eBPF and Go, and is aimed at anyone interested in diving into these technologies. We'll explore how to effectively monitor network events using eBPF, Go, and Prometheus.


## The Ghost in the Network: TCP Retransmissions

Imagine working on a high-speed, low-latency product and encountering intermittent slowdowns in data transmission. This situation can be tricky to diagnose, it is often intermittent and could bring your product to its knees. When I faced this issue, I took it upon myself to delve deep and understand what was happening under the hood. Wireshark led me to the root cause: excessive TCP retransmissions due to firewall policy.


<img width="500" alt="tcp retransmission" src="https://github.com/iogbole/ebpf-network-viz/assets/2548160/7eb67240-2514-4a4a-9140-c5c8ac603a66">


Fig 1. Depiction of TCP retransmission. Get the[ mermaid source](https://www.mermaidchart.com/raw/7d9b1dfe-a681-4079-b338-9314eed422f1?version=v0.1&theme=light&format=svg). 

TCP retransmissions aren't inherently bad; they're a fundamental part of how TCP/IP networks function. However, when they occur frequently, they can signify network issues that lead to poor application performance. A high number of retransmissions can cause:


* Increased Latency: Packets have to be sent again, which takes extra time.
* Higher CPU Usage: Both sending and receiving systems have to do additional work to handle the retransmissions.
* Bandwidth Inefficiency: Retransmissions consume bandwidth that could be better used by new data.
* User Experience Degradation: All the above contribute to a laggy or suboptimal user experience.

You can easily simulate TCP retransmission, try: 

**_sudo tc qdisc add dev eth0 root netem loss 10% delay 100ms _**

on your machine and see how it messes up your network performance and introduces high-CPU usage. I was once crazy enough to use 50% in EC2  and it booted me out of SSH connection until I restarted the node.  Do not try this out at home ;) 


## Why eBPF? 

Extended Berkeley Packet Filter (eBPF) is a revolutionary technology, available since Linux 4.x versions. Imagine eBPF as a lightweight, sandboxed virtual machine that resides within the Linux kernel, offering secure and verified access to kernel memory.

In technical terms, eBPF allows the kernel to execute BPF bytecode. The code is often written in a restricted subset of the C language, which is then compiled into BPF bytecode using a compiler like Clang. This bytecode undergoes stringent verification processes to ensure it neither intentionally nor inadvertently jeopardises the integrity of the Linux kernel. Additionally, eBPF programmes are guaranteed to execute within a finite number of instructions, making them suitable for performance-sensitive tasks such as packet filtering and network monitoring.

Functionally, eBPF allows you to run this restricted C code in response to various events, such as timers, network events, or function calls within both the kernel and user-space programs. These pieces of code are often referred to as 'probes'—kprobes for kernel function calls, uprobes for user-space function calls, and tracepoints for pre-defined hooks in the Linux kernel. In the context of this blog post, we'll be focusing on tracepoints, specifically leveraging the <code><em>tcp_retransmit_skb</em></code>  tracepoint for monitoring TCP retransmissions. Tracepoints offer a stable API for kernel observability, which is especially useful for production environments.

The flexibility, safety, and power that eBPF provides make it an invaluable tool for monitoring TCP retransmissions, a subject we will explore in detail here.

For those looking to delve deeper into eBPF, I recommend checking out the resources in the reference section, starting with [What is eBPF](https://ebpf.io/what-is-ebpf/)?

## Preparation and Environment Setup

Before we embark on our journey through code and monitoring, it's important to have your development environment properly configured. While this blog isn't an exhaustive tutorial, I'll outline the key prerequisites for your convenience.

**Using Lima on macOS**: If you're a macOS user like me, Lima is an excellent way to emulate a Linux environment. It's simple to set up and meshes seamlessly with your existing workflow. To kick things off with Lima, follow these steps:

1. Install Lima and launch it with your configuration file:

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

**Manual Setup on Linux**: If you're opting for a manual setup on Linux, here's what you'll need:

1. **Operating System**: A Linux OS like Debian is recommended, featuring the latest kernel. I'm currently using version 5.15.x. To confirm your kernel version, type `uname -a`.

2. **Linux Kernel Headers**: Ensure the Linux kernel headers are installed and reachable.

3. **Clang/LLVM**: The latest version of Clang/LLVM is crucial, as our Makefile leverages Clang for C program compilation.

4. **Libbpf, bpfcc-tools, and other bcc packages**: For the compilation process, libbpf sources are essential.  Refer to my[ ebpf-vm.yaml](https://github.com/iogbole/ebpf-network-viz/blob/main/ebpf-vm.yaml#L19) file for a list of packages. 

5. **Go**: Golang needs to be installed on your machine for this project.

With your environment now primed, you're all set to delve into the fascinating world of eBPF!


## The Solution



### Overview of Components

The code can be broadly broken down into the following components, summarised the digram below. 

1. **eBPF Code Hooks to Tracepoints**: The eBPF code uses the `tracepoint/tcp/tcp_retransmit_skb` to monitor TCP retransmissions. This allows the code to act whenever a TCP packet is retransmitted.

2. **Collect Retransmission Events**: The data related to the retransmitted packets—such as IP addresses, ports, and the protocol family—is collected in a structured manner.

3. **Bytecode Loaded by Go**: The eBPF bytecode is loaded into the kernel using a Go program, which makes use of the `github.com/cilium/ebpf` package.

4. **Use of Maps**: BPF maps are used to communicate between the eBPF code running in the kernel and the Go application running in user space.

5. **Ring Buffer in Go**: A perf event ring buffer is used in Go to read the events generated by the eBPF code.

6. **Exposed to HTTP**: The Go application exposes the metrics over HTTP on port 2112.

7. **Prometheus Scrapes Metrics**: Finally, Prometheus is configured to scrape these exposed metrics for monitoring or alerting purposes.

![ebpf-tcp drawio](https://github.com/iogbole/ebpf-network-viz/assets/2548160/915d403e-61ac-4399-9f11-d4891b32fbd4)


### Anatomy of the eBPF C Code


#### Include Headers

The headers are essential for the program to function correctly. Notably, `vmlinux.h` is a header generated by BPF CO-RE. BPF CO-RE (Compile Once, Run Everywhere) enhances the portability of eBPF programs across different kernel versions. It resolves as much as possible at compile time, using placeholders for kernel-specific information that can only be determined at runtime. When the program is loaded into the kernel, these placeholders are populated with actual values. This flexibility eliminates the need for recompilation when deploying on different kernel versions. Through BPF CO-RE, the `vmlinux.h` header is generated to represent kernel structures, making it easier to write eBPF programs that are not tightly bound to specific kernels.

```c

#include "vmlinux.h"

#include &lt;bpf/bpf_helpers.h>

#include &lt;bpf/bpf_endian.h>

#include &lt;bpf/bpf_tracing.h>

```

#### Data Structures

The `event` and `tcp_retransmit_skb_ctx` structures are defined to hold the information related to TCP retransmissions. The structures collect various fields such as timestamps, process IDs, source and destination ports, and more.

```c

struct event {

    __u64 timestamp;

    // ... other fields

};

struct tcp_retransmit_skb_ctx {

    __u64 _pad0;

    // ... other fields

};

```


##### **Finding Data Structures for Other Tracepoints**

Understanding the data structures associated with tracepoints is a key aspect when you're diving into eBPF programs for monitoring or debugging. While I focused on the `tcp_retransmit_skb` tracepoint in this blog, you may wish to explore other tracepoints. Here's how you can discover the necessary data structures for those:

1. **Locate Tracepoint Definitions**: Typically, tracepoints are defined within the Linux Kernel source code. The definitions can usually be found under `/sys/kernel/debug/tracing/events/` directory on a Linux system with the tracing subsystem enabled. Navigate through the folders to find the tracepoint of interest.

2. **Reading Format Files**: Within each tracepoint directory, you'll find a `format` file that describes the event structure. This will provide you with the types and names of the fields that are available for that particular tracepoint.

    ```

    cat /sys/kernel/debug/tracing/events/tcp/tcp_retransmit_skb/format

    ```

    This will display the format for the `tcp_retransmit_skb` tracepoint as an example.

3. **BPF Headers**: Some commonly used structures for tracepoints might already be defined in BPF-related headers (`vmlinux.h`, `bpf_helpers.h`, etc.). This saves you from needing to redefine these structures in your BPF code.

5. **Community Resources**: Sites like [eBPF.io](https://ebpf.io/) or specific GitHub repositories often have documentation or sample code that can provide additional insights into tracepoints and their associated data structures.

By familiarising yourself with the format files and possibly the kernel source code, you can create or adapt eBPF programs to tap into a wide range of system events, not just TCP retransmissions.


#### BPF Maps

The BPF map `events` is defined as a perf event array. This map serves as a communication channel between user space (Go program) and kernel space (eBPF program).

```c

struct {

    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);

    __uint(key_size, sizeof(__u32));

    __uint(value_size, sizeof(__u32));

} events SEC(".maps");

```

#### Tracepoint Function

The function `tracepoint__tcp__tcp_retransmit_skb` is attached to the `tcp_retransmit_skb` tracepoint. Here, various fields are read and stored in an `event` structure.

```c

SEC("tracepoint/tcp/tcp_retransmit_skb")

int tracepoint__tcp__tcp_retransmit_skb(struct tcp_retransmit_skb_ctx *ctx) {

    // ... code logic

}

```

#### Compiling the eBPF Code

You can compile this eBPF code using the script `run_clang.sh`:

```bash

clang -O2 -g -target bpf -c ./ebpf/retrans.c -o ./ebpf/retrans.o -I/usr/include -I/usr/src/linux-headers-$(uname -r)/include  -D __BPF_TRACING__

```


### Anatomy of the Go Program


#### Import Packages

The code starts by importing necessary Go packages including eBPF and Prometheus libraries.

```go

import (

    "github.com/cilium/ebpf"

    "github.com/prometheus/client_golang/prometheus"

    // ... other imports

)

```


#### Load eBPF Program

Here, the eBPF bytecode is loaded from the `.o` object file. I opted to load the eBPF bytecode from a pre-compiled .o object file. This object file contains the bytecode of our eBPF program, which is what gets executed within the kernel. I chose this approach to maintain a clear separation of concerns: the compilation of the eBPF program is distinct from its execution. Other examples I have seen use gobpf libraries to load the C code at compile time - this approach might be easier from a CI/CD build process. 

```go

spec, err := ebpf.LoadCollectionSpec(objFileName)

```

Attach to Tracepoint

The program attaches to the `tcp_retransmit_skb` tracepoint using the `link.Tracepoint` function.

```go

tp, err := link.Tracepoint("tcp", "tcp_retransmit_skb", prog, nil)

```

#### Perf Event Buffer

A perf event buffer is set up to read events from the kernel space.

```go

events, err := perf.NewReader(coll.Maps["events"], os.Getpagesize())

```

The Perf Event Buffer plays an essential part in bridging the gap between user-space and kernel-space communication. This buffer is a data structure that's set up to read events directly from the kernel. Essentially, it acts as a queuing mechanism, holding data that your eBPF program collects from various probes until your user-space application is ready to process it.

Here's how it generally works: 

1. Your eBPF program attaches to specific kernel functions or tracepoints and collects data, such as packet information in the case of networking or syscall information for system-level observability.

  

2. This data is then pushed to the Perf Event Buffer.

  

3. Your user-space application, written in Go, in this case, then reads from this buffer to retrieve the data for further analysis or action.

By leveraging a Perf Event Buffer, you gain a highly efficient, low-overhead mechanism for transferring data from kernel space to user space, making it a critical component in the process of monitoring, troubleshooting, and enhancing system performance. 

The Ring Buffer is a more modern alternative to Perf Event buffers, suitable for newer Kernel version

Exposing eBPF Data Through a Prometheus Endpoint

To expose the metrics gathered by your eBPF program for monitoring, we integrate Prometheus into the setup. Here's how the process works:

#### Prometheus Metrics Definition

Firstly, we define the events and metrics that Prometheus will scrape. In this instance, the focus is on TCP retransmissions. 

```go

var tcpRetransmissions = promauto.NewCounterVec(prometheus.CounterOpts{

    Name: "tcp_retransmissions_total",

    Help: "Total number of TCP retransmissions",

}, []string{"ip_version", "src_ip", "src_port", "dst_ip", "dst_port"})

```

#### Starting the HTTP Server

After defining the metrics, the next step is to expose them through an HTTP endpoint. This is done by starting an HTTP server and mapping the `/metrics` path to a Prometheus handler:

```go

// Start HTTP server for Prometheus scraping

http.Handle("/metrics", promhttp.Handler())

go func() {

    if err := http.ListenAndServe(":2112", nil); err != nil {

        panic(err)

    }

}()

```

In this example, the HTTP server listens on port 2112, and Prometheus is configured to scrape metrics from this endpoint. When Prometheus accesses the `/metrics` path, it invokes the `promhttp.Handler()`, which in turn retrieves the metric data stored in `tcpRetransmissions`. This makes the data available for Prometheus to collect, allowing for robust monitoring and alerting functionalities.

By combining these two components, you create a seamless pipeline that collects, exposes, and monitors TCP retransmission metrics in real-time.

#### The Event Loop and Metrics Update

The heart of the real-time monitoring lies in the event loop, which is continuously polling for new events from the perf event ring buffer. Each incoming event is processed and the relevant Prometheus metrics are updated accordingly.

**Listening for Events**

The loop employs the `Read()` method on the perf buffer to listen for new incoming events:

```go

for {

    record, err := events.Read()

    // ... processing code

}

```

**Event Processing and Metrics Update**

Upon receiving an event, the loop processes it and updates the Prometheus `tcpRetransmissions` metric. The specifics of this processing depend on the structure and content of the events, which are designed to capture various data fields such as timestamps, process IDs, source and destination ports, and so forth.

```go

for {

    record, err := events.Read()

    if err != nil {

        // Handle error

        continue

    }

    // Decode and process the event

    // ... processing code

    // Update Prometheus metrics

    tcpRetransmissions.WithLabelValues(/* labels */).Inc()

}

```

By marrying this event loop with the previously described Prometheus setup, the system efficiently collects, processes, and exposes metrics for TCP retransmissions in a manner that is readily compatible with other monitoring and observability tools. 

Next, ensure the go code works: 

```bash 

sudo go run retrans.go

```


This is also a good time to confirm that the Go HTTP server is up and running: 

![GO HTTPServer](https://github.com/iogbole/ebpf-network-viz/assets/2548160/35436e9e-a451-4f27-9126-5e0ecba08651)

### Setting Up Prometheus in the Lima VM using nerdctl

Since the development environment is within a Lima VM, it's advantageous to leverage nerdctl for container management. nerdctl is a Docker-compatible CLI tool for containers, which is already bundled with Lima. Here's how to set up Prometheus using a custom configuration and a shell script for automation.


#### Prometheus Configuration: `prometheus.yml`

The `prometheus.yml` configuration specifies how often Prometheus scrapes metrics and from where. In this case, it is configured to scrape the metrics exposed by the Go application running on port 2112. 

Here's the content of `prometheus.yml`:

```yaml

global:

  scrape_interval: 15s

scrape_configs:

  - job_name: 'TCPRetrans'

    static_configs:

    - targets: ['127.0.0.1:2112']

```

#### Shell Script for Automated Setup

The shell script performs several tasks to ensure Prometheus runs correctly:

1. **Getting the IP Address**: The script first retrieves the IP address of `eth0` on the host machine.

2. **Updating Configuration**: It then replaces the IP address in the `prometheus.yml` configuration file to point to the correct address where the Go application is exposing metrics.

3. **Running Prometheus**: Finally, it runs the Prometheus container using nerdctl, mapping it to port 9090.

```bash

#!/bin/bash

# nerdctl comes pre-installed with Lima.

# Lima does automatic port forwarding, so Prometheus should be accessible on your Mac at localhost:9090 when done.

# The Go app exposes an HTTP port at 2112, which should be accessible on your Mac at localhost:2112/metrics.

# Get the IP address of eth0 on the host machine.

IP_ADDRESS=$(ip -4 addr show eth0 | grep -oP '(?&lt;=inet\s)\d+(\.\d+){3}')

# Replace the IP address in the prometheus.yml file.

CONFIG_FILE="prom_config/prometheus.yml"

sed -i "s/[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+:2112/${IP_ADDRESS}:2112/g" "$CONFIG_FILE"

echo "Updated prometheus.yml with IP address: $IP_ADDRESS"

sleep 3

nerdctl run --rm -p 9090:9090 -v "$PWD/prom_config:/etc/prometheus" prom/prometheus

```

This script automates the process, making it easier to deploy Prometheus within your Lima VM. 

Since Lima also does automatic port forwarding, you should be able to access Prometheus on your Mac at `localhost:9090` and the metrics exposed by the Go application at `localhost:2112/metrics`.

Check and ensure that the job_name is registered. 


<img width="662" alt="prom_config" src="https://github.com/iogbole/ebpf-network-viz/assets/2548160/85c38250-b023-4b83-839d-2cd989a87d87">


## Create TCP Chaos: Testing It All Out

To put the eBPF program and Prometheus monitoring into action, you can introduce artificial network issues such as TCP retransmissions. The `tc` (traffic control) command in Linux allows you to simulate network issues for testing purposes. The `create_tcp_chaos.sh` shell script below automates this process, first creating the chaos and then removing it after the test.


### The `create_tcp_chaos.sh` Shell Script

Here's the script that automates the traffic control chaos:

```bash

#!/bin/bash

# Define websites to send requests to.

websites=("http://example.com" "https://www.google.com" "https://www.wikipedia.org")

# Set the number of iterations for the loop.

loop_count=20

# Introduce network latency and packet loss using tc.

sudo tc qdisc add dev eth0 root netem loss 5% delay 100ms

# Loop to send requests to the websites.

for ((i = 1; i &lt;= loop_count; i++)); do

    for site in "${websites[@]}"; do

        echo "Sending request to $site (iteration $i)"

        curl -sS "$site" > /dev/null  # s for silent and S for showing errors if they occur.

        sleep 1  # Wait for a second.

        wget -O- "$site" > /dev/null  # O- redirects output to stdout, as we don't want to save the file.

    done

done

# Remove the traffic control rule.

sudo tc qdisc del dev eth0 root

```

#### How the Script Works:

1. **Setting Up Chaos**: The script introduces packet loss and latency to `eth0` using `tc`. Specifically, it adds 5% packet loss and 100 ms delay.

2. **Testing**: It iterates through a list of websites (`example.com`, `google.com`, and `wikipedia.org`) and sends HTTP requests to them. This is done 20 times.

3. **Cleaning Up**: The script removes the `tc` rules to revert `eth0` back to its normal state.

Run the script, and you should be able to observe the effects on your Prometheus metrics. Remember to execute the script with appropriate permissions.


## Grab a coffee: Reap the Rewards

Head over to your Prometheus interface and type tcp_retransmissions_total into the query bar. Switch to the graph view and marvel at the results of your hard work.

You're now in a position to set up alerts for TCP retransmissions. A common benchmark to consider is that a retransmission rate of 2% or greater generally indicates network issues that warrant attention.

So grab a cup of coffee, sit back, and enjoy the fruit of your labour!


<img width="1510"  src="https://github.com/iogbole/ebpf-network-viz/assets/2548160/a3e79887-90d7-426d-a7e0-3054144e9738">



# Refs

* Must read - https://www.man7.org/linux/man-pages/man2/bpf.2.html
* Retrans fields: https://github.com/iovisor/bcc/blob/master/tools/tcpretrans_example.txt
* BPF CORE : https://facebookmicrosites.github.io/bpf/blog/2020/02/19/bpf-portability-and-co-re.html
* TCP tracepoints : https://www.brendangregg.com/blog/2018-03-22/tcp-tracepoints.html
* eBPF applications : https://ebpf.io/applications/
