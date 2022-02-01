package main

import (
	"fmt"
	leases "github.com/nijave/go-dhcpd-leases"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

func GenerateHostsFile(fileName string) string {
	validHostname := regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	var hostsFile strings.Builder

	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	leaseData := leases.Parse(f)
	idxIp := map[string]*leases.Lease{}
	idxHostnameIp := map[string]string{}

	// Dedupe parsed leases
	// dhcpd.leases works like a transaction log that get reconciled so items at the
	// end supersede items found earlier
	for k := range leaseData {
		lease := leaseData[k]
		ipString := lease.IP.String()
		hostname := strings.ToLower(lease.ClientHostname)
		hostname = strings.Replace(hostname, " ", "-", -1)

		if _, ipExists := idxIp[ipString]; ipExists {
			log.Printf("[delete] duplicate lease for %s\n", ipString)
			delete(idxIp, ipString)
		}

		if hostname == "" {
			log.Printf("[skip] %s with empty hostname\n", ipString)
			continue
		}

		if !validHostname.MatchString(hostname) {
			log.Printf("[skip] invalid hostname %s\n", hostname)
			continue
		}

		if lease.BindingState != "active" {
			log.Printf("[skip] %s with state %s\n", ipString, lease.BindingState)
		}

		if ip, hostnameExists := idxHostnameIp[hostname]; hostnameExists {
			log.Printf("[delete] existing lease %s with duplicate hostname %s\n", ip, hostname)
			delete(idxIp, ip)
		}

		if lease.ClientHostname != hostname {
			log.Printf("[replace] invalid hostname %s with %s\n", lease.ClientHostname, hostname)
			lease.ClientHostname = hostname
		}

		idxHostnameIp[hostname] = ipString
		idxIp[ipString] = &lease
	}

	leases := make([]*leases.Lease, 0)

	for ip := range idxIp {
		leases = append(leases, idxIp[ip])
	}

	// Sort by IP address
	sort.Slice(leases, func(i, j int) bool {
		left := leases[i].IP
		leftVal := int(left[12])<<24 + int(left[13])<<16 + int(left[14])<<8 + int(left[15])
		right := leases[j].IP
		rightVal := int(right[12])<<24 + int(right[13])<<16 + int(right[14])<<8 + int(right[15])
		return leftVal < rightVal
	})

	for _, lease := range leases {
		hostsFile.WriteString(fmt.Sprintf("%-16s %s\n", lease.IP.String(), lease.ClientHostname))
	}

	return hostsFile.String()
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	var fileName string
	if len(os.Args) == 2 {
		fileName = os.Args[1]
	} else {
		fileName = "/var/dhcpd/var/db/dhcpd.leases"
	}

	dnsmasq := DynamicPid{
		lock:    sync.Mutex{},
		pid:     -1,
		PidFile: "/var/run/dnsmasq.pid",
	}

	go http.ListenAndServe("localhost:8889", nil)

	go dnsmasq.Watch()

	dhcpWatch := KeventWatch{Filename: fileName}
	events := dhcpWatch.Watch(true)
	for {
		start := time.Now()
		log.Println("[hosts] generating new file")
		hostsFile := GenerateHostsFile(fileName)
		fd, err := os.OpenFile("/var/etc/dnsmasq-hosts-dhcp", os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("[hosts] error writing file %v\n", err)
		}
		n, err := fd.WriteString(hostsFile)
		if err != nil {
			log.Printf("[hosts] writing file %v\n", err)
		}
		log.Printf("[hosts] wrote %d bytes\n", n)
		fd.Close()
		dnsmasq.Notify(syscall.SIGHUP)
		log.Printf("[hosts] completed in %dms\n", time.Since(start).Milliseconds())

		<-events
	}
}
