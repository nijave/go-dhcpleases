package main

import (
	"log"
	"syscall"
)

type KeventWatch struct {
	Filename string
}

func (k KeventWatch) Watch() <-chan int {
	ch := make(chan int)

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
	kq, err := syscall.Kqueue()
	if err != nil {
		log.Printf("[kevent] %v\n", err)
		return nil
	}
	change := &changeBuffer[0]
	event := &eventBuffer[0]

	log.Println("[kevent] opening fd")
	fd, err := syscall.Open(k.Filename, syscall.O_RDONLY, 0)
	if err != nil {
		log.Printf("%v\n", err)
		return nil
	}

	//ev = [select.kevent(fd, filter=select.KQ_FILTER_VNODE, flags=select.KQ_EV_ADD|select.KQ_EV_ENABLE|select.KQ_EV_CLEAR, fflags=select.KQ_NOTE_WRITE|select.KQ_NOTE_EXTEND)]
	log.Println("[kevent] setting up event watch")
	syscall.SetKevent(change, fd, syscall.EVFILT_VNODE, syscall.EV_ADD|syscall.EV_ENABLE|syscall.EV_CLEAR)
	change.Fflags = syscall.NOTE_DELETE | syscall.NOTE_WRITE | syscall.NOTE_EXTEND | syscall.NOTE_ATTRIB | syscall.NOTE_RENAME | syscall.NOTE_LINK

	go func() {
		defer close(ch)
		for {
			eventNames := make([]string, 1)
			log.Println("[kqueue] waiting for an event")
			n, err := syscall.Kevent(kq, changeBuffer[:], eventBuffer[:], nil)
			if err != nil {
				log.Printf("[kqueue err %v\n", err)
				break
			}
			log.Printf("[kqueue] %d\n", n)
			if n == -1 {
				log.Printf("[kqueue] unknown error\n")
			} else if n > 0 {
				if (event.Flags & syscall.EV_ERROR) == syscall.EV_ERROR {
					log.Printf("[kqueue] errno %d\n", n)
					continue // re-open file?
				}

				for num, name := range flagNames {
					if (event.Fflags & num) == num {
						eventNames = append(eventNames, name)
					}
				}
			}
			log.Printf("[kqueue] %v\n", eventNames)
			ch <- int(event.Fflags)
		}
	}()

	return ch
}
