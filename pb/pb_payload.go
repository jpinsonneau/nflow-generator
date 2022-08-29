package pb

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"net"
	"time"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/pbflow"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	directions = []pbflow.Direction{pbflow.Direction_INGRESS, pbflow.Direction_EGRESS}
)

func GenerateRecords(ips []string) []*pbflow.Record {
	records := []*pbflow.Record{}

	t := time.Now()
	var srcIp, dstIp net.IP
	if len(ips) > 0 {
		srcIp = net.ParseIP(ips[rand.Int()%len(ips)]).To4()
		dstIp = net.ParseIP(ips[rand.Int()%len(ips)]).To4()
	} else {
		srcIp = net.ParseIP("10.10.29.7").To4()
		dstIp = net.ParseIP("10.10.29.8").To4()
	}

	flow := pbflow.Record{
		EthProtocol: rand.Uint32(),
		Direction:   directions[rand.Int()%len(directions)],
		TimeFlowStart: &timestamppb.Timestamp{
			Seconds: t.Unix(),
			Nanos:   0,
		},
		TimeFlowEnd: &timestamppb.Timestamp{
			Seconds: t.Unix(),
			Nanos:   0,
		},
		DataLink: &pbflow.DataLink{
			SrcMac: rand.Uint64(),
			DstMac: rand.Uint64(),
		},
		Network: &pbflow.Network{
			SrcAddr: &pbflow.IP{
				IpFamily: &pbflow.IP_Ipv4{
					Ipv4: ip2Long(srcIp),
				},
			},
			DstAddr: &pbflow.IP{
				IpFamily: &pbflow.IP_Ipv4{
					Ipv4: ip2Long(dstIp),
				},
			},
		},
		Transport: &pbflow.Transport{
			SrcPort:  uint32(rand.Int() % 9999),
			DstPort:  uint32(rand.Int() % 9999),
			Protocol: uint32(rand.Int() % 255),
		},
		Bytes:     uint64(getRandInt(1, 9000)),
		Packets:   1,
		Interface: "fake nflow-generator record",
	}
	records = append(records, &flow)

	return records
}

func ip2Long(ip net.IP) uint32 {
	var long uint32
	binary.Read(bytes.NewBuffer(ip), binary.BigEndian, &long)
	return long
}

func getRandInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}
