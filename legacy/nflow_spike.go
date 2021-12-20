package legacy

import "log"

//Generate a netflow packet w/ user-defined record count
func GenerateSpike(spikeProto string) Netflow {
	data := new(Netflow)
	data.Header = CreateNFlowHeader(1)
	data.Records = spikeFlowPayload(spikeProto)
	return *data
}

func spikeFlowPayload(spikeProto string) []NetflowPayload {
	payload := make([]NetflowPayload, 1)
	switch spikeProto {
	case "ssh":
		payload[0] = CreateSshFlow()
	case "ftp":
		payload[0] = CreateFTPFlow()
	case "http":
		payload[0] = CreateHttpFlow()
	case "https":
		payload[0] = CreateHttpsFlow()
	case "ntp":
		payload[0] = CreateNtpFlow()
	case "snmp":
		payload[0] = CreateSnmpFlow()
	case "imaps":
		payload[0] = CreateImapsFlow()
	case "mysql":
		payload[0] = CreateMySqlFlow()
	case "https_alt":
		payload[0] = CreateHttpAltFlow()
	case "p2p":
		payload[0] = CreateP2pFlow()
	case "bittorrent":
		payload[0] = CreateBitorrentFlow()
	default:
		log.Fatalf("protocol option %s is not valid, see --help for options", spikeProto)
	}
	return payload
}
