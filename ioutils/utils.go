package ioutils

import (
	"net"
)

func CopyTcp(dest net.Conn, src net.Conn) {
	data := make([]byte, 1024)
	for {
		n, err1 := src.Read(data)
		_, err2 := dest.Write(data[:n])
		if err2 != nil || err1 != nil {
			dest.Close()
			src.Close()
			break
		}
	}
}
