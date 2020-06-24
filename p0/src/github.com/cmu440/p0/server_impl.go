// Implementation of a MultiEchoServer. Students should write their code in this file.

package p0

//允许使用的包 bufio fmt net strconv

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type multiEchoServer struct {
	ln            net.Listener
	clients       map[*client]bool
	opChan        chan bool
	broadcastChan chan []byte
}

type client struct {
	cn      net.Conn
	readBuf []byte
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	s := &multiEchoServer{
		broadcastChan: make(chan []byte, 100),
		clients: make(map[*client]bool),
		opChan: make(chan bool, 1),
	}
	s.opChan <- true
	return s
}

func (mes *multiEchoServer) Start(port int) error {
	addr := ":" + strconv.FormatInt(int64(port), 10)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		ptErr(err)
		return err
	}
	mes.ln = l
	go mes.Handle()
	go mes.BroadCast()
	return nil
}

func (mes *multiEchoServer) Handle() {
	for {
		conn, err := mes.ln.Accept()
		if err != nil {
			//ptErr(err)
			continue
		}
		c := &client{cn: conn}
		mes.addConn(c)
		go mes.handle(c)
	}
}

func (mes *multiEchoServer) addConn(c *client) {
	<-mes.opChan
	defer func() { mes.opChan <- true }()
	mes.clients[c] = true
}

func (mes *multiEchoServer) killConn(t *client) {
	<-mes.opChan
	defer func() { mes.opChan <- true }()
	t.cn.Close()
	delete(mes.clients, t)
}

func (mes *multiEchoServer) handle(c *client) {
	reader := bufio.NewReader(c.cn)
	for {
		// Read up to and including the first '\n' character.
		msgBytes, err := reader.ReadBytes('\n')
		if err != nil {
			mes.killConn(c)
			//ptErr(err)
			return
		}
		mes.broadcastChan<-msgBytes
	}
	/*
	for {
		var b [1024]byte
		n, err := c.cn.Read(b[:])
		c.cn.SetReadDeadline(time.Now().Add(time.Second))
		if err != nil {
			switch {
			case err == io.EOF:
				mes.broadcastChan <- c.readBuf
				c.readBuf = nil
				continue
			case strings.Contains(err.Error(), "timeout"):
				mes.killConn(c)
				return
			default:
				ptErr(err)
			}
			continue
		}
		c.readBuf = append(c.readBuf, b[:n]...)
	}
	 */
}

func (mes *multiEchoServer) BroadCast() {
	for {
		select {
		case b := <-mes.broadcastChan:
			func() {
				<-mes.opChan
				defer func() { mes.opChan <- true }()
				for client := range mes.clients {
					client.cn.SetWriteDeadline(time.Now().Add(time.Second))
					_, err := client.cn.Write(b)
					if err == nil {
						continue
					}
					if strings.Contains(err.Error(), "broken pipe") {
						mes.killConn(client)
						continue
					}
					if strings.Contains(err.Error(), "connection reset by peer") {
						mes.killConn(client)
						continue
					}
					ptErr(err)

				}
			}()
		}
	}
}

func (mes *multiEchoServer) Close() {
	// TODO: implement this!
}

func (mes *multiEchoServer) Count() int {
	<-mes.opChan
	defer func() { mes.opChan <- true }()
	return len(mes.clients)
}

// TODO: add additional methods/functions below!

func ptErr(err error) {
	fmt.Println("err:", err)
}
