CC := clang
CFLAGS := -O2 -g -target bpf -I/usr/include -I/usr/src/linux-headers-$(shell uname -r)/include -D__BPF_TRACING__
SRC := ./src/ebpf/retrans.c
OBJ := ./src/ebpf/retrans.o
VMLINUX_H := ./src/ebpf/vmlinux.h

all: build_ebpf build_go

build_ebpf: $(VMLINUX_H) $(OBJ)

$(VMLINUX_H):
	bpftool btf dump file /sys/kernel/btf/vmlinux format c > $(VMLINUX_H)

$(OBJ): $(SRC) $(VMLINUX_H)
	$(CC) $(CFLAGS) -c $< -o $@ || (echo "Error building eBPF program"; exit 1)  # Stop on error

build_go:
	sudo go run ./src/main.go || (echo "Error running Go program"; exit 1)  # Stop on error

clean:
	rm -f $(OBJ) 

.PHONY: all build_ebpf build_go clean
