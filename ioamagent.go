package ioamagent

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	ioamapi "github.com/jbiggley/clt-go"
)

const (
	ETH_P_IPV6          = 0x86DD
	IPV6_TLV_IOAM       = 49
	IOAM_PREALLOC_TRACE = 0

	TRACE_TYPE_BIT0_MASK  = 1 << 23 // Hop_Lim + Node Id (short)
	TRACE_TYPE_BIT1_MASK  = 1 << 22 // Ingress/Egress Ids (short)
	TRACE_TYPE_BIT2_MASK  = 1 << 21 // Timestamp seconds
	TRACE_TYPE_BIT3_MASK  = 1 << 20 // Timestamp fraction
	TRACE_TYPE_BIT4_MASK  = 1 << 19 // Transit Delay
	TRACE_TYPE_BIT5_MASK  = 1 << 18 // Namespace Data (short)
	TRACE_TYPE_BIT6_MASK  = 1 << 17 // Queue depth
	TRACE_TYPE_BIT7_MASK  = 1 << 16 // Checksum Complement
	TRACE_TYPE_BIT8_MASK  = 1 << 15 // Hop_Lim + Node Id (wide)
	TRACE_TYPE_BIT9_MASK  = 1 << 14 // Ingress/Egress Ids (wide)
	TRACE_TYPE_BIT10_MASK = 1 << 13 // Namespace Data (wide)
	TRACE_TYPE_BIT11_MASK = 1 << 12 // Buffer Occupancy
	TRACE_TYPE_BIT22_MASK = 1 << 1  // Opaque State Snapshot
)

// parseNodeData extracts IOAM node data from the packet based on the trace type
func parseNodeData(p []byte, ttype uint32) (*ioamapi.IOAMNode, error) {
	node := &ioamapi.IOAMNode{}
	i := 0

	if ttype&TRACE_TYPE_BIT11_MASK != 0 {
		node.BufferOccupancy = binary.BigEndian.Uint32(p[i : i+4])
		i += 4
	}

	if ttype&TRACE_TYPE_BIT22_MASK != 0 {
		opaqueLen := p[i]
		node.OSS = &ioamapi.OpaqueStateSnapshot{
			SchemaId: uint32(binary.BigEndian.Uint32(p[i:i+4]) & 0x00FFFFFF),
		}
		i += 4

		if opaqueLen > 0 {
			node.OSS.Data = make([]byte, opaqueLen*4)
			copy(node.OSS.Data, p[i:i+opaqueLen*4])
		}
		i += opaqueLen * 4
	}

	return node, nil
}

// parseIOAMTrace extracts IOAM trace data from the packet
func parseIOAMTrace(p []byte) (*ioamapi.IOAMTrace, error) {
	// Extract relevant fields from the packet
	ns, nodelen, _, remlen, ttype, _, tid, sid := unpackPacket(p[:32])

	nodes := []*ioamapi.IOAMNode{}
	i := 32 + remlen*4

	for i < len(p) {
		node, err := parseNodeData(p[i:i+nodelen*4], ttype)
		if err != nil {
			return nil, err
		}
		i += nodelen * 4

		// Handle TRACE_TYPE_BIT22_MASK if present
		if ttype&TRACE_TYPE_BIT22_MASK != 0 {
			opaqueLen := p[i]
			node.OSS = &ioamapi.OpaqueStateSnapshot{
				SchemaId: uint32(binary.BigEndian.Uint32(p[i:i+4]) & 0x00FFFFFF),
			}
			if opaqueLen > 0 {
				node.OSS.Data = p[i+4 : i+4+int(opaqueLen)*4]
			}
			i += 4 + int(opaqueLen)*4
		}

		nodes = append(nodes, node)
	}

	trace := &ioamapi.IOAMTrace{
		BitField:    ttype << 8,
		NamespaceId: ns,
		TraceIdHigh: tid >> 64,
		TraceIdLow:  tid & 0x0000000000000000FFFFFFFFFFFFFFFF,
		SpanId:      sid,
		Nodes:       nodes,
	}

	return trace, nil
}

func main() {
	interfaceName := flag.String("i", "", "Interface to listen on")
	output := flag.Bool("o", false, "Output traces to stdout")
	collector := flag.String("c", "", "IOAM collector address")
	flag.Parse()

	if *interfaceName == "" {
		fmt.Println("Error: Interface not specified")
		flag.Usage()
		os.Exit(1)
	}

	// Set up the socket to listen for IOAM packets
	conn, err := net.ListenPacket("ip6:"+fmt.Sprintf("%04x", ETH_P_IPV6), *interfaceName)
	if err != nil {
		fmt.Printf("Error: Failed to open socket on interface %s\n", *interfaceName)
		os.Exit(1)
	}
	defer conn.Close()

	var stub ioamapi.IOAMServiceClient
	if !*output {
		if *collector == "" {
			fmt.Println("Error: IOAM collector address not specified")
			flag.Usage()
			os.Exit(1)
		}

		cc, err := grpc.Dial(*collector, grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Error: Failed to connect to IOAM collector at %s\n", *collector)
			os.Exit(1)
		}
		defer cc.Close()

		stub = ioamapi.NewIOAMServiceClient(cc)
	}

	buf := make([]byte, 2048)

	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Printf("Error: Failed to read packet: %v\n", err)
			os.Exit(1)
		}

		trace, err := parseIOAMTrace(buf[:n])
		if err != nil {
			fmt.Printf("Error: Failed to parse packet: %v\n", err)
			continue
		}

		if *output {
			fmt.Println(trace)
		} else {
			_, err := stub.Report(context.Background(), trace)
			if err != nil {
				fmt.Printf("Error: Failed to report trace to IOAM collector: %v\n", err)
			}
		}
	}
}
