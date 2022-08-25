package pb

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/grpc"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/pbflow"
)

var glog = logrus.WithField("component", "exporter/GRPCProto")

// GRPCProto flow exporter. Its ExportFlows method accepts slices of *pbflow.Record
// by its input channel and submits them to the collector.
type GRPCProto struct {
	hostPort   string
	clientConn *grpc.ClientConnection
}

func StartGRPCProto(hostPort string) (*GRPCProto, error) {
	clientConn, err := grpc.ConnectClient(hostPort)
	if err != nil {
		return nil, err
	}
	return &GRPCProto{
		hostPort:   hostPort,
		clientConn: clientConn,
	}, nil
}

// ExportFlows accepts slices of *pbflow.Record by its input channel,
//and submits them to the collector.
func (g *GRPCProto) ExportFlows(input <-chan []*pbflow.Record) {
	log := glog.WithField("collector", g.hostPort)
	for inputRecords := range input {
		entries := make([]*pbflow.Record, 0, len(inputRecords))
		entries = append(entries, inputRecords...)
		log.Debugf("sending %d records", len(entries))
		if _, err := g.clientConn.Client().Send(context.TODO(), &pbflow.Records{
			Entries: entries,
		}); err != nil {
			log.WithError(err).Error("couldn't send flow records to collector")
		}
	}
	if err := g.clientConn.Close(); err != nil {
		log.WithError(err).Warn("couldn't close flow export client")
	}
}
