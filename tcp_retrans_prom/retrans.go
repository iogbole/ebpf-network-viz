package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	objFileName = "./ebpf/retrans.o"
)

type tcpRetransmitEvent struct {
	Timestamp uint64
	PID       uint32
	Sport     uint16
	Dport     uint16
	Saddr     [4]byte
	Daddr     [4]byte
	SaddrV6   [16]byte
	DaddrV6   [16]byte
	Family    uint16
	State     int32
}

var tcpRetransmissions = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "tcp_retransmissions_total",
	Help: "Total number of TCP retransmissions",
}, []string{"ip_version", "src_ip", "src_port", "dst_ip", "dst_port"})

func main() {
	// Load eBPF program
	spec, err := ebpf.LoadCollectionSpec(objFileName)
	if err != nil {
		panic(err)
	}

	coll, err := ebpf.NewCollectionWithOptions(spec, ebpf.CollectionOptions{
		Programs: ebpf.ProgramOptions{
			//Verbose to catch eBPF verifier issues
			LogLevel: 1,
			LogSize:  65535,
		},
	})
	if err != nil {
		panic(err)
	}

	prog := coll.Programs["tracepoint__tcp__tcp_retransmit_skb"]
	if prog == nil {
		panic("Failed to find tracepoint__tcp__tcp_retransmit_skb program")
	}

	// Attach the program to the tcp_retransmit_skb tracepoint
	tp, err := link.Tracepoint("tcp", "tcp_retransmit_skb", prog, nil)
	if err != nil {
		panic(err)
	}
	defer tp.Close()

	// Set up the perf ring buffer to receive events
	events, err := perf.NewReader(coll.Maps["events"], os.Getpagesize())
	if err != nil {
		panic(err)
	}
	defer events.Close()

	// Set up signal handling
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	// Start HTTP server for Prometheus scraping
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":2112", nil); err != nil {
			panic(err)
		}
	}()

	// Listen for events from the perf ring buffer
	fmt.Println("Monitoring TCP retransmissions...")
	for {
		select {
		case <-sig:
			fmt.Println("\nReceived signal, stopping...")
			return
		default:
			record, err := events.Read()
			if err != nil {
				if perf.IsUnknownEvent(err) {
					continue
				}
				panic(err)
			}

			event := tcpRetransmitEvent{}
			err = binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event)
			if err != nil {
				panic(err)
			}

			timestamp := time.Unix(0, int64(event.Timestamp)).Format(time.RFC3339)
			var srcIP, dstIP string
			ipVersion := 0
			if event.Family == 2 { // AF_INET
				ipVersion = 4
				srcIP = fmt.Sprintf("%d.%d.%d.%d", event.Saddr[0], event.Saddr[1], event.Saddr[2], event.Saddr[3])
				dstIP = fmt.Sprintf("%d.%d.%d.%d", event.Daddr[0], event.Daddr[1], event.Daddr[2], event.Daddr[3])
			} else if event.Family == 10 { // AF_INET6
				ipVersion = 6
				srcIP = fmt.Sprintf("%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x",
					event.SaddrV6[0], event.SaddrV6[1], event.SaddrV6[2], event.SaddrV6[3],
					event.SaddrV6[4], event.SaddrV6[5], event.SaddrV6[6], event.SaddrV6[7],
					event.SaddrV6[8], event.SaddrV6[9], event.SaddrV6[10], event.SaddrV6[11],
					event.SaddrV6[12], event.SaddrV6[13], event.SaddrV6[14], event.SaddrV6[15])
				dstIP = fmt.Sprintf("%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x",
					event.DaddrV6[0], event.DaddrV6[1], event.DaddrV6[2], event.DaddrV6[3],
					event.DaddrV6[4], event.DaddrV6[5], event.DaddrV6[6], event.DaddrV6[7],
					event.DaddrV6[8], event.DaddrV6[9], event.DaddrV6[10], event.DaddrV6[11],
					event.DaddrV6[12], event.DaddrV6[13], event.DaddrV6[14], event.DaddrV6[15])
			}

			tcpRetransmissions.WithLabelValues(
				strconv.Itoa(ipVersion), srcIP, strconv.Itoa(int(event.Sport)),
				dstIP, strconv.Itoa(int(event.Dport)),
			).Inc()

			output := map[string]interface{}{
				"timestamp": timestamp,
				"pid":       event.PID,
				"state":     event.State,
				"ipversion": ipVersion,
				"source": map[string]interface{}{
					"ip":   srcIP,
					"port": event.Sport,
				},
				"destination": map[string]interface{}{
					"ip":   dstIP,
					"port": event.Dport,
				},
			}
			jsonOutput, err := json.Marshal(output)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(jsonOutput))
		}
	}
}
