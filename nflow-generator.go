package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	ipfix "nflow-generator/ipfix"
	leg "nflow-generator/legacy"
	"os"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
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
	CollectorIP   string `short:"t" long:"target" description:"target ip address of the netflow collector"`
	CollectorPort string `short:"p" long:"port" description:"port number of the target netflow collector"`
	SpikeProto    string `short:"s" long:"spike" description:"run a second thread generating a spike for the specified protocol"`
	FalseIndex    bool   `short:"f" long:"false-index" description:"generate false SNMP interface indexes, otherwise set to 0"`
	IPs           string `short:"i" long:"ips" description:"use specific list of ips, comma separated"`
	V10           bool   `short:"v" long:"ipfix" description:"use ipfix version (10)"`
	Help          bool   `short:"h" long:"help" description:"show nflow-generator help"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		showUsage()
		os.Exit(1)
	}
	if opts.Help == true {
		showUsage()
		os.Exit(1)
	}
	if opts.CollectorIP == "" || opts.CollectorPort == "" {
		showUsage()
		os.Exit(1)
	}
	collector := opts.CollectorIP + ":" + opts.CollectorPort
	udpAddr, err := net.ResolveUDPAddr("udp", collector)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal("Error connecting to the target collector: ", err)
	}
	log.Infof("sending netflow data to a collector ip: %s and port: %s",
		opts.CollectorIP, opts.CollectorPort)

	var ips []string
	if len(opts.IPs) > 0 {
		ips = strings.Split(opts.IPs, ",")
		log.Info("specified ips:")
		for _, ip := range ips {
			log.Infof("%s", ip)
		}
	}

	var byteArray []byte
	for {
		rand.Seed(time.Now().Unix())
		n := leg.RandomNum(50, 1000)

		if opts.V10 {
			msg := ipfix.GenerateNetflow(ips)
			js, _ := json.Marshal(msg)
			fmt.Println(bytes.NewBuffer(js).String())
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
		fmt.Println(byteArray)
		_, err := conn.Write(byteArray)
		if err != nil {
			log.Fatal("Error connecting to the target collector: ", err)
		}

		// add some periodic spike data
		if n < 150 {
			sleepInt := time.Duration(3000)
			time.Sleep(sleepInt * time.Millisecond)
		}
		sleepInt := time.Duration(n)
		time.Sleep(sleepInt * time.Millisecond)
	}
}

func showUsage() {
	var usage string
	usage = `
Usage:
  main [OPTIONS] [collector IP address] [collector port number]

  Send mock Netflow version 5 data to designated collector IP & port.
  Time stamps in all datagrams are set to UTC.

Application Options:
  -t, --target= target ip address of the netflow collector
  -p, --port=   port number of the target netflow collector
  -s, --spike run a second thread generating a spike for the specified protocol
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
  -f, --false-index generate a false snmp index values of 1 or 2. The default is 0. (Optional)
  -i, --ips use specific list of ips, comma separated (Optional)
  -v, --ipfix use ipfix version (10)

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
