package main

import (
	"io"
	"log"
	"net"
)

func listen(name string, localAddr string, remoteAddr string) {
	localTcpAddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		log.Printf("[%s] Can not resolve local TCP Address %s: %s", name, localAddr, err)
		return
	}

	localSock, err := net.ListenTCP("tcp", localTcpAddr)
	if err != nil || localSock == nil {
		log.Printf("[%s] Can not listen on %s: %s", name, localTcpAddr, err)
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
			log.Printf("[%s] Connection from %s", name, conn.RemoteAddr())
		}
		go forward(name, conn, remoteAddr)
	}
}

func forward(name string, local net.Conn, remoteAddr string) {
	sourceAddr := local.RemoteAddr()
	defer func() {
		log.Printf("[%s] Disconnected: %s", name, sourceAddr)
	}()

	d1 := make(chan struct{})
	d2 := make(chan struct{})

	remote, err := net.Dial("tcp", remoteAddr)
	if err != nil || remote == nil {
		log.Printf("[%s] Failed to dial to remote %s: %s", name, remoteAddr, err)
		local.Close()
		return
	}

	doCopy := func(in, out net.Conn, done chan struct{}) {
		io.Copy(out, in)
		out.Close()
		close(done)
	}
	go doCopy(local, remote, d1)
	go doCopy(remote, local, d2)

	<-d1
	<-d2
}
