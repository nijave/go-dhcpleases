package main

import (
	"log"
	"os"
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
	events := pidFileWatch.Watch(false)

	for {
		log.Printf("[pid] reading %s\n", p.PidFile)
		fd, err := os.Open(p.PidFile)
		if err != nil {
			log.Printf("[pid] error %v opening pid file %s\n", err, p.PidFile)
			time.Sleep(1 * time.Second)
			continue
		}

		n, err := fd.Read(data)
		fd.Close()
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
