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

func (c *collector) Report(grpc_ctx context.Context, request *ioam_api.IOAMTrace) (*emptypb.Empty, error) {
	var traceID trace.TraceID
	binary.BigEndian.PutUint64(traceID[:8], request.GetTraceId_High())
	binary.BigEndian.PutUint64(traceID[8:], request.GetTraceId_Low())

	var spanID trace.SpanID
	binary.BigEndian.PutUint64(spanID[:], request.GetSpanId())

	span_ctx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), span_ctx)

	tracer := otel.Tracer("ioam-tracer")
	_, span := tracer.Start(ctx, "ioam-span")
	defer span.End()

	i := 1
	for _, node := range request.GetNodes() {
		nsID := strconv.FormatUint(uint64(request.GetNamespaceId()), 10)
		key := "ioam_namespace" + nsID + "_node" + strconv.Itoa(i)
		str := c.parseNode(node, request.GetBitField())

		span.SetAttributes(attribute.String(key, str))
		i += 1
	}

	return empty_inst, nil
}

func (c *collector) parseNode(node *ioam_api.IOAMNode, fields uint32) string {
str := ""
if (fields & MASK_BIT0) != 0 {
	str += "HopLimit=" + strconv.FormatUint(uint64(node.GetHopLimit()), 10) + "; "
	str += "Id=" + strconv.FormatUint(uint64(node.GetId()), 10) + "; "
}
if (fields & MASK_BIT1) != 0 {
	str += "IngressId=" + strconv.FormatUint(uint64(node.GetIngressId()), 10) + "; "
	str += "EgressId=" + strconv.FormatUint(uint64(node.GetEgressId()), 10) + "; "
}
if (fields & MASK_BIT2) != 0 {
	str += "TimestampSecs=" + strconv.FormatUint(uint64(node.GetTimestampSecs()), 10) + "; "
}
if (fields & MASK_BIT3) != 0 {
	str += "TimestampFrac=" + strconv.FormatUint(uint64(node.GetTimestampFrac()), 10) + "; "
}
if (fields & MASK_BIT4) != 0 {
	str += "TransitDelay=" + strconv.FormatUint(uint64(node.GetTransitDelay()), 10) + "; "
}
if (fields & MASK_BIT5) != 0 {
	str += "NamespaceData=0x" + hex.EncodeToString(node.GetNamespaceData()) + "; "
}
if (fields & MASK_BIT6) != 0 {
	str += "QueueDepth=" + strconv.FormatUint(uint64(node.GetQueueDepth()), 10) + "; "
}
if (fields & MASK_BIT7) != 0 {
	str += "CsumComp=" + strconv.FormatUint(uint64(node.GetCsumComp()), 10) + "; "
}
if (fields & MASK_BIT8) != 0 {
	str += "HopLimit=" + strconv.FormatUint(uint64(node.GetHopLimit()), 10) + "; "
	str += "IdWide=" + strconv.FormatUint(uint64(node.GetIdWide()), 10) + "; "
}
if (fields & MASK_BIT9) != 0 {
	str += "IngressIdWide=" + strconv.FormatUint(uint64(node.GetIngressIdWide()), 10) + "; "
	str += "EgressIdWide=" + strconv.FormatUint(uint64(node.GetEgressIdWide()), 10) + "; "
}
if (fields & MASK_BIT10) != 0 {
	str += "NamespaceDataWide=0x" + hex.EncodeToString(node.GetNamespaceDataWide()) + "; "
}
if (fields & MASK_BIT11) != 0 {
	str += "BufferOccupancy=" + strconv.FormatUint(uint64(node.GetBufferOccupancy()), 10) + "; "
}
if (fields & MASK_BIT22) != 0 {
	str += "OpaqueStateSchemaId=" + strconv.FormatUint(uint64(node.GetOSS().GetSchemaId()), 10) + "; "
	str += "OpaqueStateData=0x" + hex.EncodeToString(node.GetOSS().GetData()) + "; "
}

return str
}
