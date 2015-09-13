package fwd

import (
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Listener struct {
	name       string
	localAddr  string
	remoteAddr string
	firstTry   bool
	stop       chan struct{}
}

type Forwarder struct {
	mutex     *sync.Mutex
	listeners map[string]*Listener
}

func NewForwarder() *Forwarder {
	f := &Forwarder{
		mutex:     &sync.Mutex{},
		listeners: make(map[string]*Listener, 10),
	}
	return f
}

func (f *Forwarder) TryListen(name string, localAddr string, remoteAddr string, retry bool) {
	f.mutex.Lock()
	if l, found := f.listeners[name]; found {
		l.Stop()
	}
	l := NewListener(name, localAddr, remoteAddr)
	f.listeners[name] = l
	f.mutex.Unlock()

	for {
		l.Listen()
		if retry {
			time.Sleep(10 * time.Second)
		} else {
			return
		}
	}
}

func NewListener(name string, localAddr string, remoteAddr string) *Listener {
	l := &Listener{
		name:       name,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		firstTry:   true,
		stop:       make(chan struct{}),
	}
	return l
}

func (l *Listener) Listen() {
	defer func() {
		l.firstTry = false
	}()
	localTcpAddr, err := net.ResolveTCPAddr("tcp", l.localAddr)
	if err != nil {
		if l.firstTry {
			log.Printf("[%s] Can not resolve local TCP Address %s: %s", l.name, l.localAddr, err)
		}
		return
	}

	localSock, err := net.ListenTCP("tcp", localTcpAddr)
	if err != nil || localSock == nil {
		if l.firstTry {
			log.Printf("[%s] Can not listen on %s: %s", l.name, localTcpAddr, err)
		}
		return
	} else {
		log.Printf("[%s] Bridging %s to %s", l.name, localTcpAddr, l.remoteAddr)
	}

	for {
		select {
		case <-l.stop:
			return
		default:
		}
		conn, err := localSock.Accept()
		if err != nil || conn == nil {
			log.Printf("[%s] Failed to accept connection: %s", l.name, err)
			continue
		} else {
			log.Printf("[%s] Connection from %s started", l.name, conn.RemoteAddr())
		}
		go l.forward(conn)
	}
}

func (l *Listener) forward(local net.Conn) {
	sourceAddr := local.RemoteAddr()

	t0 := time.Now()

	var bytesIn int64
	var bytesOut int64

	defer func() {
		log.Printf("[%s] Connection from %s finished after %s (In: %d, Out: %d)", l.name, sourceAddr, time.Since(t0), bytesIn, bytesOut)
	}()

	d1 := make(chan struct{})
	d2 := make(chan struct{})

	remote, err := net.Dial("tcp", l.remoteAddr)
	if err != nil || remote == nil {
		log.Printf("[%s] Failed to dial to remote %s: %s", l.name, l.remoteAddr, err)
		local.Close()
		return
	}

	doCopy := func(in, out net.Conn, done chan struct{}, bytes *int64) {
		written, _ := io.Copy(out, in)
		atomic.AddInt64(bytes, written)
		out.Close()
		close(done)
	}
	go doCopy(local, remote, d1, &bytesOut)
	go doCopy(remote, local, d2, &bytesIn)

	// wait until both directions are finished
	<-d1
	<-d2
}

func (f *Forwarder) Stop() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, l := range f.listeners {
		l.Stop()
	}
}

func (l *Listener) Stop() {
	l.stop <- struct{}{}
}
