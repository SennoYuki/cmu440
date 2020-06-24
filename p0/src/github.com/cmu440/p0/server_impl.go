// Implementation of a MultiEchoServer. Students should write their code in this file.

package p0

//You may only use the following packages: bufio, fmt, net, and strconv.
//因为net.Listen->Accept得到的connection默认读写都不超时，没想到不使用time包实现超时丢弃消息的方法
import (
	"bufio"
	"net"
	"strconv"
	"time"
)

type multiEchoServer struct {
	ln            net.Listener
	clients       map[*client]bool
	opChan        chan bool
	broadcastChan chan []byte
}

type client struct {
	cn net.Conn
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	s := &multiEchoServer{
		broadcastChan: make(chan []byte, 100),
		clients:       make(map[*client]bool),
		opChan:        make(chan bool, 1),
	}
	s.opChan <- true
	return s
}

func (mes *multiEchoServer) Start(port int) error {
	addr := ":" + strconv.FormatInt(int64(port), 10)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	mes.ln = l
	go mes.Handle()
	go mes.BroadCast()
	return nil
}

func (mes *multiEchoServer) Close() {
	// TODO: implement this!
}

func (mes *multiEchoServer) Count() int {
	<-mes.opChan
	defer func() { mes.opChan <- true }()
	return len(mes.clients)
}

// additional methods/functions

func (mes *multiEchoServer) Handle() {
	for {
		conn, err := mes.ln.Accept()
		if err != nil {
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
			return
		}
		mes.broadcastChan <- msgBytes
	}
}

func (mes *multiEchoServer) BroadCast() {
	for {
		select {
		case b := <-mes.broadcastChan:
			func() {
				<-mes.opChan
				defer func() { mes.opChan <- true }()
				for client := range mes.clients {
					//超时停止写入，b可能会写入一部分，不影响后续操作
					client.cn.SetWriteDeadline(time.Now().Add(time.Second))
					client.cn.Write(b)
				}
			}()
		}
	}
}
