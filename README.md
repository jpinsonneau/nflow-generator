# Usage - nflow-generator

This program generates mock netflow (v5/v10) data that can be used to test netflow collectors. 
This is a fork from [nflow-generator](https://github.com/nerdalert/nflow-generator) with implementation of ipfix from [ipfix-gen](https://github.com/cang233/ipfix-gen) and [vflow](https://github.com/EdgeCast/vflow)

### Build

Install [Go](http://golang.org/doc/install), then:
```bash
git clone https://github.com/netobserv/nflow-generator
cd nflow-generator
go build
```	

### Run
Feed it the target collector and port, and optional "false-index" flag:
```bash
./nflow-generator -t <ip> -p <port> [ -f | --false-index ]
```

### Help
Use `-h` option to get all applications options and usage examples:
```bash
./nflow-generator -h
```
### Deploy on Kubernetes

Edit file [netflow_generator.yaml](./examples/netflow_generator.yaml) `<collector_ip>` and `<collector_port>`.
Add extra parameters if needed (ips for example)

Then run:
```bash
kubectl apply -f examples/netflow_generator.yaml
```