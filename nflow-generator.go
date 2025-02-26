package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"nflow-generator/ipfix"
	"nflow-generator/legacy"
	"nflow-generator/pb"
	"os"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/grpc"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/pbflow"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type Proto int

const (
	FTP Proto = iota + 1
	SSH
	DNS
	HTTP
	HTTPS
	NTP
	SNMP
	IMAPS
	MYSQL
	HTTPS_ALT
	P2P
	BITTORRENT
)

var opts struct {
	CollectorIPs  string `short:"t" long:"targets" description:"target ip address(es) the netflow collector(s), comma separated"`
	CollectorPort int    `short:"p" long:"port" description:"port number of the target netflow collector. Default 2055"`
	SpikeProto    string `long:"spike" description:"run a second thread generating a spike for the specified protocol"`
	FalseIndex    bool   `long:"false-index" description:"generate false SNMP interface indexes, otherwise set to 0"`
	IPs           string `short:"i" long:"ips" description:"use specific list of ips, comma separated"`
	Type          string `long:"type" description:"use 'legacy' for netflow v5, 'ipfix' for v10 or 'pb' for fake ebpf agent. Default is legacy"`
	Sleep         bool   `short:"s" long:"sleep" description:"enable random sleep time"`
	MinSleep      int    `long:"minsleep" description:"min sleep time. Default: 50"`
	MaxSleep      int    `long:"maxsleep" description:"max sleep time. Default: 1000"`
	RateSleep     int    `long:"ratesleep" description:"sleep time between each rate log. Default: 10"`
	Concurrency   int    `long:"concurrency" description:"number of threads to run in parallel"`
	Help          bool   `short:"h" long:"help" description:"show nflow-generator help"`
}

var err error
var ips []string
var collectorAddrs []*net.UDPAddr
var loopCount float64 = 0

func main() {
	_, err = flags.Parse(&opts)

	if err != nil {
		showUsage()
		os.Exit(1)
	}

	if opts.Help {
		showUsage()
		os.Exit(1)
	}

	if opts.CollectorIPs == "" {
		showUsage()
		os.Exit(1)
	}

	if opts.CollectorPort == 0 {
		opts.CollectorPort = 2055
	}

	if opts.MinSleep == 0 {
		opts.MinSleep = 50
	}

	if opts.MaxSleep == 0 {
		opts.MaxSleep = 1000
	}

	if opts.RateSleep == 0 {
		opts.RateSleep = 10
	}

	splittedCollectorIPsString := strings.Split(opts.CollectorIPs, ",")
	for _, ip := range splittedCollectorIPsString {
		if !strings.Contains(ip, ":") {
			ip = fmt.Sprintf("%s:%d", ip, opts.CollectorPort)
		}

		log.Infof("checking collector: %s", ip)

		collectorAddr, err := net.ResolveUDPAddr("udp", ip)
		if err != nil {
			log.Fatal(err)
		}

		collectorAddrs = append(collectorAddrs, collectorAddr)
	}

	if len(opts.IPs) > 0 {
		splittedIPsString := strings.Split(opts.IPs, ",")
		log.Info("specified ips:")
		for _, ip := range splittedIPsString {
			block := ipaddr.NewIPAddressString(ip).GetAddress()
			for i := block.Iterator(); i.HasNext(); {
				ip := i.Next().GetNetIPAddr().String()
				ips = append(ips, ip)
				log.Infof("%s", ip)
			}
		}
	}

	if opts.Concurrency == 0 {
		opts.Concurrency = 1
	}

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < opts.Concurrency; i++ {
		go loopFlows()
	}

	loopRate()
}

func loopFlows() {
	i := rand.Int() % len(collectorAddrs)

	var grpcConn *grpc.ClientConnection
	var flows []*pbflow.Record

	var udpConn *net.UDPConn
	var byteArray []byte

	target := fmt.Sprintf("%s:%d", collectorAddrs[i].IP.String(), collectorAddrs[i].Port)
	if opts.Type == "pb" {
		log.Infof("checking grpc target %s ...", target)

		grpcConn, err = grpc.ConnectClient(target)
		if err != nil {
			log.Fatal("Error resolving grpcExporter: ", err)
		}
		log.Infof("grpc target %s ok !", target)
	} else {
		log.Infof("checking udp target %s ...", target)

		addr, err := net.ResolveUDPAddr("udp", target)
		if err != nil {
			log.Fatal("Error resolving udp address: ", err)
		}
		udpConn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Fatal("Error dialing udp address: ", err)
		}
		log.Infof("udp target %s ok !", target)
	}

	for {
		n := legacy.RandomNum(opts.MinSleep, opts.MaxSleep)

		switch opts.Type {
		case "ipfix":
			msg := ipfix.GenerateNetflow(ips)
			byteArray = ipfix.Encode(*msg, ipfix.GetSeqNum())
		case "pb":
			flows = pb.GenerateRecords(ips)
		default:
			// add spike data
			if opts.SpikeProto != "" {
				legacy.GenerateSpike(opts.SpikeProto)
			}
			recordCount := 16
			if n > 900 {
				recordCount = 8
			}
			data := legacy.GenerateNetflow(recordCount, ips, opts.FalseIndex)
			byteArray = legacy.BuildNFlowPayload(data)
		}

		if grpcConn != nil {
			_, err = grpcConn.Client().Send(context.TODO(), &pbflow.Records{
				Entries: flows,
			})
		} else if udpConn != nil {
			_, err = udpConn.Write(byteArray)
		} else {
			err = errors.New("either grpc or udp connection should be set")
		}

		if err != nil {
			log.Fatal("Error connecting to the target collector: ", err)
		}

		if opts.Sleep {
			// add some periodic spike data
			if n < 150 {
				sleepInt := time.Duration(3000)
				time.Sleep(sleepInt * time.Millisecond)
			}
			sleepInt := time.Duration(n)
			time.Sleep(sleepInt * time.Millisecond)
		}

		loopCount++
	}
}

func loopRate() {
	for {
		loopCount = 0

		time.Sleep(time.Duration(opts.RateSleep) * time.Second)

		rate := loopCount / float64(opts.RateSleep)
		log.Infof("Current rate is: %.1f calls per seconds", rate)
	}
}

func showUsage() {
	usage := `
Usage:
  main [OPTIONS] [collector IP address] [collector port number]

  Send mock Netflow version 5 data to designated collector IP & port.
  Time stamps in all datagrams are set to UTC.

Application Options:
  -t, --targets= target ip address(es) the netflow collector(s), comma separated
  -p, --port=   port number of the target netflow collector. Default 2055
  --spike run a second thread generating a spike for the specified protocol
    protocol options are as follows:
        ftp - generates tcp/21
        ssh  - generates tcp/22
        dns - generates udp/54
        http - generates tcp/80
        https - generates tcp/443
        ntp - generates udp/123
        snmp - generates ufp/161
        imaps - generates tcp/993
        mysql - generates tcp/3306
        https_alt - generates tcp/8080
        p2p - generates udp/6681
        bittorrent - generates udp/6682
  --false-index generate a false snmp index values of 1 or 2. The default is 0. (Optional)
  -i, --ips use specific list of ips, comma separated (Optional)
  --type use 'legacy' for netflow v5, 'ipfix' for v10 or 'pb' for fake ebpf agent. Default is legacy
  -s, --sleep enable random sleep time
	--minsleep min sleep time. Default: 50
	--maxsleep max sleep time. Default: 1000
	--ratesleep sleep time between each rate log. Default: 10
	--concurrency number of threads to run in parallel

Example Usage:

    -first build from source (one time)
    go build   

    -generate default flows to device 172.16.86.138, port 9995
    ./nflow-generator -t 172.16.86.138 -p 9995 

    -generate default flows between ips 172.16.86.1, 172.16.86.2, 172.16.86.3 to device 172.16.86.138, port 9995
    ./nflow-generator -t 172.16.86.138 -p 9995 -i 172.16.86.1,172.16.86.2,172.16.86.3

    -generate default flows along with a spike in the specified protocol:
    ./nflow-generator -t 172.16.86.138 -p 9995 -s ssh

    -generate default flows with "false index" settings for snmp interfaces 
    ./nflow-generator -t 172.16.86.138 -p 9995 -f

Help Options:
  -h, --help    Show this help message
  `
	fmt.Print(usage)
}
