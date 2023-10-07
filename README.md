# Learning eBPF

> **Warning**
>This repo does not follow the typical Go project layout. Every folder in this repo has a main function, as it is solely intended to demonstrate various possibilities.


## Dev env setup 

There's a [Lima](https://github.com/lima-vm/lima) config file with the packages needed for building the code.

Install lima, then: 

```
limactl start ebpf-vm.yaml
limactl shell ebpf-vm
```

If you'd like to use Visual Studio Code, 

Get the SSH command 

`limactl show-ssh ebpf-vm` 

Then [Connect to remote server via SSH](https://code.visualstudio.com/docs/remote/ssh) in Visual Studio Code

Next, clone the repo 

`git clone https://github.com/iogbole/ebpf-playground.git`


##  Running TCP retransmit ebpf code 

```
cd ebpf-playground

cd tcp_retransmit 

make #to run the make file
```


Or run each step below manually. The Makefile above automates all the steps below.



```

sudo apt-get install -y bpfcc-tools #should be installed as part of the lima startup 

bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h # See tip 2 below.

./clang.sh  # Compile C code 

go build retrans.go # Build go code  

./retrans # run go code 

```

## Simulate packet loss to cause TCP transmission 

Now run the ./curl.sh to simulate packet loss. This script generates multiple TCP events using curl and wget, but `5%` of the TCP events are corrupted by force.

`sudo tc qdisc add dev eth0 root netem loss 5% delay 100ms`

You may increase the `5%` to `10%` if you want to force the kernel to perform more retransmissions, but doing so may disconnect your SSH access and the HTTP listener in the Go app.



## The Prometheus example 

the `tcp_retrans_prom` folder contains an example of how to expose the telemetry for Prom to scrape it. 

Execute the `run_prom.sh` to get prom started in the lima VM. 
On your mac, go to http://localhost:9090 to be sure it is up and running 

The go app runs an HTTP server for prom at http://localhost:2112 


## Prom Output 
http://locahost:9090 


<img width="1410" alt="Screenshot 2023-04-23 at 7 45 30 pm" src="https://user-images.githubusercontent.com/2548160/233858880-d68090ce-26aa-48f7-b698-46275ade0e31.png">

<img width="1404" alt="Screenshot 2023-04-23 at 7 43 37 pm" src="https://user-images.githubusercontent.com/2548160/233858885-0e011398-f7a4-47ee-8809-c1e2156402af.png">


## Console Output 

```
{"destination":{"ip":"142.250.180.4","port":443},"ipversion":4,"pid":0,"source":{"ip":"192.168.5.15","port":38130},"state":65536,"timestamp":"1970-01-04T23:40:37Z"}
{"destination":{"ip":"142.250.180.4","port":443},"ipversion":4,"pid":0,"source":{"ip":"192.168.5.15","port":38130},"state":720896,"timestamp":"1970-01-04T23:40:38Z"}
{"destination":{"ip":"142.250.180.4","port":443},"ipversion":4,"pid":0,"source":{"ip":"192.168.5.15","port":60288},"state":65536,"timestamp":"1970-01-04T23:40:44Z"}
{"destination":{"ip":"192.168.5.2","port":51121},"ipversion":4,"pid":0,"source":{"ip":"192.168.5.15","port":22},"state":65536,"timestamp":"1970-01-04T23:41:13Z"}
{"destination":{"ip":"142.250.180.4","port":443},"ipversion":4,"pid":0,"source":{"ip":"192.168.5.15","port":56912},"state":65536,"timestamp":"1970-01-04T23:41:51Z"}
{"destination":{"ip":"142.250.180.4","port":443},"ipversion":4,"pid":0,"source":{"ip":"192.168.5.15","port":56912},"state":65536,"timestamp":"1970-01-04T23:41:52Z"}
```
This output indicates that a TCP retransmission event has been captured, and it provides detailed of the event. 

--


## Tips 

1. Display tracepoint return fields 
`sudo cat /sys/kernel/debug/tracing/events/tcp/tcp_retransmit_skb/format`

2. To create the vmlinux.h file, you will need to use the BPF CO-RE (Compile Once, Run Everywhere) feature provided by bpftool. The vmlinux.h file is a generated header file that includes kernel structures and definitions required for BPF programs.

To generate the vmlinux.h file, follow these steps:

First, ensure you have bpftool installed on your system. You can usually find it in the linux-tools package or compile it from the kernel source.

```
bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h
```

Now you should have a vmlinux.h file in your current working directory. You can include this file in your eBPF C programs to access kernel structures and definitions.

Please note that the  vmlinux.h file is specific to the kernel version and configuration, so it's recommended to generate it for each target system where you want to run your eBPF program.


## Refs

Must read - https://www.man7.org/linux/man-pages/man2/bpf.2.html 
Retrans fields: https://github.com/iovisor/bcc/blob/master/tools/tcpretrans_example.txt
BPF CORE : https://facebookmicrosites.github.io/bpf/blog/2020/02/19/bpf-portability-and-co-re.html 
TCP tracepoints : https://www.brendangregg.com/blog/2018-03-22/tcp-tracepoints.html 
eBPF applications : https://ebpf.io/applications/
