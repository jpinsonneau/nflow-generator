package main

import (
	"fmt"
	"math/rand"
	"net"
	ipfix "nflow-generator/ipfix"
	leg "nflow-generator/legacy"
	"os"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
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
	CollectorPort string `short:"p" long:"port" description:"port number of the target netflow collector"`
	SpikeProto    string `long:"spike" description:"run a second thread generating a spike for the specified protocol"`
	FalseIndex    bool   `long:"false-index" description:"generate false SNMP interface indexes, otherwise set to 0"`
	IPs           string `short:"i" long:"ips" description:"use specific list of ips, comma separated"`
	V10           bool   `short:"v" long:"ipfix" description:"use ipfix version (10)"`
	Sleep         bool   `short:"s" long:"sleep" description:"enable random sleep time"`
	MinSleep      int    `long:"minsleep" description:"min sleep time"`
	MaxSleep      int    `long:"maxsleep" description:"max sleep time"`
	Concurrency   int    `long:"concurrency" description:"number of threads to run in parallel"`
	Help          bool   `short:"h" long:"help" description:"show nflow-generator help"`
}

var start time.Time
var ips []string
var udpAddrs []*net.UDPAddr
var loopCount float64 = 0

func main() {
	_, err := flags.Parse(&opts)

	if err != nil {
		showUsage()
		os.Exit(1)
	}

	if opts.Help {
		showUsage()
		os.Exit(1)
	}

	if opts.CollectorIPs == "" || opts.CollectorPort == "" {
		showUsage()
		os.Exit(1)
	}

	if opts.MinSleep == 0 {
		opts.MinSleep = 50
	}

	if opts.MaxSleep == 0 {
		opts.MaxSleep = 1000
	}

	splittedCollectorIPsString := strings.Split(opts.CollectorIPs, ",")
	for _, ip := range splittedCollectorIPsString {
		collector := ip + ":" + opts.CollectorPort
		log.Infof("checking collector: %s", collector)

		udpAddr, err := net.ResolveUDPAddr("udp", collector)
		if err != nil {
			log.Fatal(err)
		}

		udpAddrs = append(udpAddrs, udpAddr)
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

	start = time.Now()
	rand.Seed(start.Unix())
	for i := 0; i < opts.Concurrency; i++ {
		go loopFlows()
	}

	loopRate()
}

func loopFlows() {
	i := rand.Int() % len(udpAddrs)

	log.Infof("sending netflow data to collector %s:%d using %d threads",
		udpAddrs[i].IP, udpAddrs[i].Port, opts.Concurrency)

	conn, err := net.DialUDP("udp", nil, udpAddrs[i])
	if err != nil {
		log.Fatal("Error connecting to the target collector: ", err)
	}

	var byteArray []byte
	for {
		n := leg.RandomNum(opts.MinSleep, opts.MaxSleep)

		if opts.V10 {
			msg := ipfix.GenerateNetflow(ips)
			byteArray = ipfix.Encode(*msg, ipfix.GetSeqNum())
		} else {
			// add spike data
			if opts.SpikeProto != "" {
				leg.GenerateSpike(opts.SpikeProto)
			}
			recordCount := 16
			if n > 900 {
				recordCount = 8
			}
			data := leg.GenerateNetflow(recordCount, ips, opts.FalseIndex)
			byteArray = leg.BuildNFlowPayload(data)
		}
		_, err := conn.Write(byteArray)
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
		time.Sleep(10 * time.Second)

		now := time.Now()
		diff := now.Sub(start).Seconds()
		rate := loopCount / diff
		log.Infof("Current rate is: %f calls per seconds", rate)
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
  -p, --port=   port number of the target netflow collector
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
  -v, --ipfix use ipfix version (10)
  -s, --sleep enable random sleep time
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
