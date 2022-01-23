package main

import (
	"fmt"
	leases "github.com/npotts/go-dhcpd-leases"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type DynamicPid struct {
	lock    sync.Mutex
	pid     int
	PidFile string
}

func (p DynamicPid) GetPid() int {
	var pid int
	p.lock.Lock()
	pid = p.pid
	p.lock.Unlock()
	return pid
}

func (p DynamicPid) SetPid(pid int) {
	p.lock.Lock()
	p.pid = pid
	p.lock.Unlock()
}

func (p DynamicPid) Watch() {
	data := make([]byte, 8)
	pidFileWatch := KeventWatch{Filename: p.PidFile}
	events := pidFileWatch.Watch()

	for {
		log.Printf("[pid] reading %s\n", p.PidFile)
		fd, err := os.Open(p.PidFile)
		if err != nil {
			log.Printf("[pid] error %v opening pid file %s\n", err, p.PidFile)
			time.Sleep(1 * time.Second)
			continue
		}

		n, err := fd.Read(data)
		if err != nil {
			log.Printf("[pid] error %v reading pid file %s\n", err, p.PidFile)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("[pid] read %d bytes from %s\n", n, p.PidFile)

		pidText := strings.TrimSpace(string(data[0:n]))
		pid, err := strconv.Atoi(pidText)
		if err != nil {
			log.Printf("[pid] couldn't parse pid %v `%s`\n", data, pidText)
			time.Sleep(1 * time.Second)
			continue
		}
		p.SetPid(pid)

		log.Printf("[pid] watching pid file %s for changes\n", p.PidFile)
		<-events
	}
}

func (p DynamicPid) Notify(signal syscall.Signal) bool {
	if p.GetPid() < 1 {
		return false
	}

	err := syscall.Kill(p.GetPid(), signal)
	if err != nil {
		log.Printf("[notify] %v notifying dnsmasq\n", err)
		return false
	}

	return true
}

func GenerateHostsFile(fileName string) {
	validHostname := regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	log := log.New(os.Stderr, "", 0)

	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}

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
		fmt.Printf("%-16s %s\n", lease.IP.String(), lease.ClientHostname)
	}
}

func main() {
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

	go dnsmasq.Watch()

	dhcpWatch := KeventWatch{Filename: fileName}
	for _ = range dhcpWatch.Watch() {
		GenerateHostsFile(fileName)
		dnsmasq.Notify(syscall.SIGHUP)
	}
}
