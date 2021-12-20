package ipfix

import (
	"encoding/binary"
	"math"
	"math/rand"
	"net"
	"time"
)

var initialTemplateId = uint16(257)
var templateID uint16 = initialTemplateId

func GetTemplateID() uint16 {
	templateID++
	if templateID >= math.MaxUint16-1 {
		templateID = initialTemplateId
	}
	return templateID
}

//check rfc5102_model file for ids
func GetIDs() []uint16 {
	return []uint16{
		4,  //protocolIdentifier
		7,  //sourceTransportPort
		8,  //sourceIPv4Address
		56, //sourceMacAddress
		11, //destinationTransportPort
		12, //destinationIPv4Address
		80, //destinationMacAddress
		21, //flowEndSysUpTime

	}
}

func GetVals(ips []string) []interface{} {
	var srcIp, dstIp net.IP
	if len(ips) > 0 {
		srcIp = net.ParseIP(ips[rand.Int()%len(ips)]).To4()
		dstIp = net.ParseIP(ips[rand.Int()%len(ips)]).To4()
	} else {
		srcIp = net.ParseIP("10.10.29.7").To4()
		dstIp = net.ParseIP("10.10.29.8").To4()
	}

	t := time.Now()
	srcMac, _ := net.ParseMAC("2F-F3-40-59-B0-CC")
	dstMac, _ := net.ParseMAC("8B-83-A4-83-76-41")
	return []interface{}{
		[]byte{34},
		HostTo2Net(1234),
		srcIp,
		srcMac,
		HostTo2Net(5678),
		dstIp,
		dstMac,
		HostTo4Net(uint32(t.UnixNano())),
	}
}

var initialSequenceNumber = uint32(234234234)
var seqNum uint32 = initialSequenceNumber

func GetSeqNum() uint32 {
	seqNum++
	if seqNum >= math.MaxUint32-1 {
		seqNum = initialSequenceNumber
	}
	return seqNum
}

func GenerateNetflow(ips []string) *Message {
	ids := GetIDs()
	vals := GetVals(ips)
	templateID := GetTemplateID()
	var fields []FieldSpecifier
	var dfs []DataField
	for i := 0; i < len(ids); i++ {
		fields = append(fields, FieldSpecifier{
			ID:           ids[i],
			Length:       uint16(InfoModel[ElementKey{0, ids[i]}].Type.minLen()),
			EnterpriseNo: 0,
		})
		dfs = append(dfs, DataField{
			FieldID: ids[i],
			Value:   vals[i],
		})
	}

	return &Message{
		Header: MessageHeader{
			Version:    VERSION,
			Length:     0,
			ExportTime: 0,
			SequenceNo: 0,
			DomainID:   0,
		},
		TemplateSet: []TemplateSet{
			{
				Header: SetHeader{
					ID:     2,
					Length: 0,
				},
				Templates: []TemplateRecord{{
					ID:         templateID,
					FieldCount: uint16(len(ids)),
					Fields:     fields,
				}},
			},
		},
		OptionsTemplateSet: nil,
		DataSet: []DataSet{
			{
				Header: SetHeader{
					ID:     templateID,
					Length: 0,
				},
				DataFields: dfs,
			},
		},
	}
}

func HostTo2Net(n uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, n)
	return b
}

func HostTo4Net(n uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return b
}
