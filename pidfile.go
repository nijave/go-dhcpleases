package main

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type DynamicPid struct {
	pidAvailable chan int
	pid          *int64
	signals      chan syscall.Signal
	PidFile      string
}

func (p DynamicPid) GetPid() int {
	return int(atomic.LoadInt64(p.pid))
}

func (p DynamicPid) setPid(pid int) {
	atomic.StoreInt64(p.pid, int64(pid))
}

func StartWatcher(pidFile string) *DynamicPid {
	emptyPid := int64(-1)
	watcher := &DynamicPid{
		pidAvailable: make(chan int, 1),
		pid:          &emptyPid,
		signals:      make(chan syscall.Signal, 1),
		PidFile:      pidFile,
	}

	go watcher.watch()
	go watcher.notifier()

	return watcher
}

func (p DynamicPid) watch() {
	data := make([]byte, 8)
	pidFileWatch := KeventWatch{Filename: p.PidFile}
	fileWatchEvents := pidFileWatch.Watch(false)

	for {
		log.Printf("[pid] reading %s", p.PidFile)
		fd, err := os.Open(p.PidFile)
		if err != nil {
			log.Printf("[pid] error %v opening pid file %s", err, p.PidFile)
			time.Sleep(1 * time.Second)
			continue
		}

		n, err := fd.Read(data)
		fd.Close()
		if err != nil {
			log.Printf("[pid] error %v reading pid file %s", err, p.PidFile)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("[pid] read %d bytes from %s", n, p.PidFile)

		pidText := strings.TrimSpace(string(data[0:n]))
		pid, err := strconv.Atoi(pidText)
		if err != nil {
			log.Printf("[pid] couldn't parse pid %v `%s`", data, pidText)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("[pid] read pid=%d from %s", pid, p.PidFile)
		p.setPid(pid)

		p.pidAvailable <- pid

		log.Printf("[pid] watching pid file %s for changes", p.PidFile)
		<-fileWatchEvents
	}
}

func (p DynamicPid) notifier() {
	log.Println("[pid] notifier waiting on pid")
	<-p.pidAvailable
	log.Printf("[pid] notifier got pid=%d", p.GetPid())

	if p.GetPid() == -1 {
		log.Println("[pid] err pid is still -1")
		return
	}

	var signal syscall.Signal
	for {
		signal = <-p.signals
		log.Printf("[signal] send signum=%d pid=%d", signal, p.GetPid())
		err := syscall.Kill(p.GetPid(), signal)
		if err != nil {
			log.Printf("[notify] %v notifying pid=%d", err, p.GetPid())
		}
	}
}

func (p DynamicPid) Notify(signal syscall.Signal) {
	log.Printf("[signal] queue signum=%d pid=%d", signal, p.GetPid())
	p.signals <- signal
}
