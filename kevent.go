package main

import (
	"strings"
	"syscall"
	"time"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type KeventWatch struct {
	Filename string
}

func (k KeventWatch) Watch(block bool) <-chan int {
	var buf int
	if block {
		buf = 16
	} else {
		buf = 1
	}
	ch := make(chan int, buf)

	flagNames := map[uint32]string{
		syscall.NOTE_DELETE: "NOTE_DELETE",
		syscall.NOTE_WRITE:  "NOTE_WRITE",
		syscall.NOTE_EXTEND: "NOTE_EXTEND",
		syscall.NOTE_ATTRIB: "NOTE_ATTRIB",
		syscall.NOTE_LINK:   "NOTE_LINK",
		syscall.NOTE_RENAME: "NOTE_RENAME",
		syscall.NOTE_REVOKE: "NOTE_REVOKE",
	}

	var changeBuffer [1]syscall.Kevent_t
	var eventBuffer [1]syscall.Kevent_t
	change := &changeBuffer[0]
	event := &eventBuffer[0]

	go func() {
		defer close(ch)
		failures := 0
		for {
			if failures > 10 {
				log.Printf("[kevent] too many failures encountered. aborting")
				break
			}

			kq, err := syscall.Kqueue()
			if err != nil {
				failures += 1
				log.Printf("[kevent] %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			log.Printf("[kevent] opening fd %s", k.Filename)
			fd, err := syscall.Open(k.Filename, syscall.O_RDONLY, 0)
			if err != nil {
				failures += 1
				log.Printf("%v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log.Printf("[kevent] fd=%d opened for %s", fd, k.Filename)

			log.Printf("[kevent] fd=%d setting up event watch", fd)
			syscall.SetKevent(change, fd, syscall.EVFILT_VNODE, syscall.EV_ADD|syscall.EV_ENABLE|syscall.EV_CLEAR)
			change.Fflags = syscall.NOTE_DELETE | syscall.NOTE_WRITE | syscall.NOTE_EXTEND | syscall.NOTE_ATTRIB | syscall.NOTE_RENAME | syscall.NOTE_LINK

			for {
				eventNames := make([]string, 1)
				log.Printf("[kqueue] fd=%d waiting for an event", fd)
				n := -1
				for n == -1 {
					n, err = syscall.Kevent(kq, changeBuffer[:], eventBuffer[:], nil)
					if n == -1 {
						log.Printf("[kqueue] fd=%d syscall.kevent -> EINTR", fd)
					}
				}

				if (event.Flags & syscall.EV_ERROR) == syscall.EV_ERROR {
					log.Printf("[kqueue] fd=%d errno %d %v", fd, n, err)
					break // re-open file
				}

				for num, name := range flagNames {
					if (event.Fflags & num) == num {
						eventNames = append(eventNames, name)
					}
				}

				log.Printf("[kqueue] fd=%d [%s]", fd, strings.Join(eventNames, ", "))
				select {
				case ch <- int(event.Flags):
				default:
					if block {
						ch <- int(event.Flags)
					} else {
						log.Printf("[kqueue] fd=%d notify channel full, skipping", fd)
					}
				}

				failures = max(0, failures-1)

				if (event.Fflags & syscall.NOTE_DELETE) == syscall.NOTE_DELETE {
					log.Printf("[kevent] fd=%d file deleted. re-opening", fd)
					break // re-open file
				}
			}

			syscall.Close(int(fd))
		}
	}()

	return ch
}
