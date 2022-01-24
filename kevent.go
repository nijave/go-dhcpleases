package main

import (
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
		buf = 0
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
				log.Printf("[kevent] too many failures encountered. aborting\n")
				break
			}

			kq, err := syscall.Kqueue()
			if err != nil {
				failures += 1
				log.Printf("[kevent] %v\n", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			log.Println("[kevent] opening fd")
			fd, err := syscall.Open(k.Filename, syscall.O_RDONLY, 0)
			if err != nil {
				failures += 1
				log.Printf("%v\n", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			log.Println("[kevent] setting up event watch")
			syscall.SetKevent(change, fd, syscall.EVFILT_VNODE, syscall.EV_ADD|syscall.EV_ENABLE|syscall.EV_CLEAR)
			change.Fflags = syscall.NOTE_DELETE | syscall.NOTE_WRITE | syscall.NOTE_EXTEND | syscall.NOTE_ATTRIB | syscall.NOTE_RENAME | syscall.NOTE_LINK

			for {
				eventNames := make([]string, 1)
				log.Println("[kqueue] waiting for an event")
				n := -1
				for n == -1 {
					n, err = syscall.Kevent(kq, changeBuffer[:], eventBuffer[:], nil)
					if n == -1 {
						log.Printf("[kqueue] syscall.kevent -> EINTR\n", err)
					}
				}

				if (event.Flags & syscall.EV_ERROR) == syscall.EV_ERROR {
					log.Printf("[kqueue] errno %d %v\n", n)
					break // re-open file
				}

				for num, name := range flagNames {
					if (event.Fflags & num) == num {
						eventNames = append(eventNames, name)
					}
				}

				log.Printf("[kqueue] %v\n", eventNames)
				select {
				case ch <- int(event.Flags):
				default:
					if block {
						ch <- int(event.Flags)
					} else {
						log.Printf("[kqueue] notify channel full, skipping\n")
					}
				}

				failures = max(0, failures-1)
			}
		}
	}()

	return ch
}
