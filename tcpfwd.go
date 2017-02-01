package main

import (
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
)

func tryListen(name string, localAddr string, remoteAddr string, retry bool) {
	firstTry := true
	for {
		listen(name, localAddr, remoteAddr, firstTry)
		firstTry = false
		if retry {
			time.Sleep(10 * time.Second)
		} else {
			return
		}
	}
}

func listen(name string, localAddr string, remoteAddr string, firstTry bool) {
	localTcpAddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		if firstTry {
			log.Printf("[%s] Can not resolve local TCP Address %s: %s", name, localAddr, err)
		}
		return
	}

	localSock, err := net.ListenTCP("tcp", localTcpAddr)
	if err != nil || localSock == nil {
		if firstTry {
			log.Printf("[%s] Can not listen on %s: %s", name, localTcpAddr, err)
		}
		return
	} else {
		log.Printf("[%s] Bridging %s to %s", name, localTcpAddr, remoteAddr)
	}

	for {
		conn, err := localSock.Accept()
		if err != nil || conn == nil {
			log.Printf("[%s] Failed to accept connection: %s", name, err)
			continue
		} else {
			log.Printf("[%s] Connection from %s started", name, conn.RemoteAddr())
		}
		go forward(name, conn, remoteAddr)
	}
}

func forward(name string, local net.Conn, remoteAddr string) {
	sourceAddr := local.RemoteAddr()

	t0 := time.Now()

	var bytesIn int64
	var bytesOut int64

	defer func() {
		Bytes.WithLabelValues(name, "in").Add(float64(bytesIn))
		Bytes.WithLabelValues(name, "out").Add(float64(bytesOut))
		log.Printf("[%s] Connection from %s finished after %s (In: %d, Out: %d)", name, sourceAddr, time.Since(t0), bytesIn, bytesOut)
	}()

	d1 := make(chan struct{})
	d2 := make(chan struct{})

	remote, err := net.Dial("tcp", remoteAddr)
	if err != nil || remote == nil {
		log.Printf("[%s] Failed to dial to remote %s: %s", name, remoteAddr, err)
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

	Conns.WithLabelValues(name).Inc()
	// wait until both directions are finished
	<-d1
	<-d2
}
