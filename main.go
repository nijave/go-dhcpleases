package main

import (
	"flag"
	"fmt"
	leases "github.com/nijave/go-dhcpd-leases"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

func GenerateHostsFile(fileName string, domainSuffix string) string {
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

		if strings.Contains(lease.ClientHostname, ".") {
			log.Printf("[strip] suffix from %s\n", lease.ClientHostname)
			lease.ClientHostname = strings.Split(lease.ClientHostname, ".")[0]
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
		hostsFile.WriteString(fmt.Sprintf("%-16s", lease.IP.String()))
		if domainSuffix != "" {
			hostsFile.WriteString(fmt.Sprintf(" %s.%s", lease.ClientHostname, domainSuffix))
		}
		hostsFile.WriteString(fmt.Sprintf(" %s", lease.ClientHostname))
		hostsFile.WriteString("\n")
	}

	return hostsFile.String()
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	leaseFilePath := flag.String("l", "/var/dhcpd/var/db/dhcpd.leases", "lease file path")
	dnsmasqPidFilePath := flag.String("p", "/var/run/dnsmasq.pid", "dnsmasq pid file path")
	noProfile := flag.Bool("no-profile", false, "disable http profile server")
	domainSuffix := flag.String("d", "", "domain suffix to append to host entries")

	flag.Parse()

	if !*noProfile {
		go http.ListenAndServe("localhost:8889", nil)
	}

	dnsmasq := StartWatcher(*dnsmasqPidFilePath)

	dhcpWatch := KeventWatch{Filename: *leaseFilePath}
	events := dhcpWatch.Watch(true)
	for {
		start := time.Now()
		log.Println("[hosts] generating new file")
		hostsFile := GenerateHostsFile(*leaseFilePath, *domainSuffix)

		fd, err := os.OpenFile("/var/etc/dnsmasq-hosts-dhcp", os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("[hosts] error writing file %v\n", err)
		}
		n, err := fd.WriteString(hostsFile)
		if err != nil {
			log.Printf("[hosts] writing file %v\n", err)
		}
		fd.Close()
		log.Printf("[hosts] wrote %d bytes\n", n)

		dnsmasq.Notify(syscall.SIGHUP)
		log.Printf("[hosts] completed in %dms\n", time.Since(start).Milliseconds())

		<-events
	}
}
