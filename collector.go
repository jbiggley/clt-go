package collector

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/grpc"
	
	"github.com/jbiggley/clt-go/ioam_api"
)

type collector struct {
	mask map[uint32]string
}

var (
	empty_inst = new(emptypb.Empty)
)

func NewCollector() *collector {
	return &collector{
		mask: map[uint32]string{
			1 << 31: "HopLimit=%d; Id=%d; ",
			1 << 30: "IngressId=%d; EgressId=%d; ",
			1 << 29: "TimestampSecs=%d; ",
			1 << 28: "TimestampFrac=%d; ",
			1 << 27: "TransitDelay=%d; ",
			1 << 26: "NamespaceData=0x%s; ",
			1 << 25: "QueueDepth=%d; ",
			1 << 24: "CsumComp=%d; ",
			1 << 23: "HopLimit=%d; IdWide=%d; ",
			1 << 22: "IngressIdWide=%d; EgressIdWide=%d; ",
			1 << 21: "NamespaceDataWide=0x%s; ",
			1 << 20: "BufferOccupancy=%d; ",
			1 << 9:  "OpaqueStateSchemaId=%d; OpaqueStateData=0x%s; ",
		},
	}
}

func (c *collector) Report(ctx context.Context, request *ioam_api.IOAMTrace) (*emptypb.Empty, error) {
	traceID := trace.TraceID{}
	binary.BigEndian.PutUint64(traceID[:8], request.GetTraceId_High())
	binary.BigEndian.PutUint64(traceID[8:], request.GetTraceId_Low())

	spanID := trace.SpanID{}
	binary.BigEndian.PutUint64(spanID[:], request.GetSpanId())

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx = trace.ContextWithSpanContext(ctx, spanCtx)

	tracer := otel.Tracer("ioam-tracer")
	_, span := tracer.Start(ctx, "ioam-span")
	defer span.End()

	i := 1
	for _, node := range request.GetNodes() {
		nsID := strconv.FormatUint(uint64(request.GetNamespaceId()), 10)
		key := "ioam_namespace" + nsID + "_node" + strconv.Itoa(i)
		str := c.parseNode(node, request.GetBitField())

		span.SetAttributes(attribute.String(key, str))
		i++
	}

	return empty_inst, nil
}


type field struct {
	name   string
	format string
	value  uint64
}

func (f field) toString() string {
	return fmt.Sprintf(f.format, f.value)
}

func (c *collector) parseNode(node *ioam_api.IOAMNode, fields uint32) string {
	var fieldsToParse = []field{
		{name: "HopLimit", format: "HopLimit=%d; ", value: uint64(node.GetHopLimit())},
		{name: "Id", format: "Id=%d; ", value: uint64(node.GetId())},
		{name: "IngressId", format: "IngressId=%d; ", value: uint64(node.GetIngressId())},
		{name: "EgressId", format: "EgressId=%d; ", value: uint64(node.GetEgressId())},
		{name: "TimestampSecs", format: "TimestampSecs=%d; ", value: uint64(node.GetTimestampSecs())},
		{name: "TimestampFrac", format: "TimestampFrac=%d; ", value: uint64(node.GetTimestampFrac())},
		{name: "TransitDelay", format: "TransitDelay=%d; ", value: uint64(node.GetTransitDelay())},
		{name: "NamespaceData", format: "NamespaceData=0x%s; ", value: uint64(node.GetNamespaceData())},
		{name: "QueueDepth", format: "QueueDepth=%d; ", value: uint64(node.GetQueueDepth())},
		{name: "CsumComp", format: "CsumComp=%d; ", value: uint64(node.GetCsumComp())},
		{name: "IdWide", format: "IdWide=%d; ", value: uint64(node.GetIdWide())},
		{name: "IngressIdWide", format: "IngressIdWide=%d; ", value: uint64(node.GetIngressIdWide())},
		{name: "EgressIdWide", format: "EgressIdWide=%d; ", value: uint64(node.GetEgressIdWide())},
		{name: "NamespaceDataWide", format: "NamespaceDataWide=0x%s; ", value: uint64(node.GetNamespaceDataWide())},
		{name: "BufferOccupancy", format: "BufferOccupancy=%d; ", value: uint64(node.GetBufferOccupancy())},
		{name: "OpaqueStateSchemaId", format: "OpaqueStateSchemaId=%d; ", value: uint64(node.GetOSS().GetSchemaId())},
		{name: "OpaqueStateData", format: "OpaqueStateData=0x%s; ", value: uint64(node.GetOSS().GetData())},
	}

	var str strings.Builder
	for _, f := range fieldsToParse {
		if (fields & c.mask[1<<31]) != 0 {
			str.WriteString(f.toString())
		}
	}
	return str.String()
}

return str
}
