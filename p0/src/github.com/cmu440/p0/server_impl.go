// Implementation of a MultiEchoServer. Students should write their code in this file.

package p0

//You may only use the following packages: bufio, fmt, net, and strconv.
//因为net.Listen->Accept得到的connection默认读写都不超时，没想到不使用time包实现超时丢弃消息的方法
import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
)

type multiEchoServer struct {
	ln      net.Listener
	clients map[*client]bool
	opLock  *ConcurrencyResource
}

type client struct {
	cn net.Conn
	wr chan []byte
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	s := &multiEchoServer{
		clients: make(map[*client]bool),
		opLock:  NewConcurrencyResource(1),
	}
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
	return nil
}

func (mes *multiEchoServer) Close() {
	// TODO: implement this!
}

func (mes *multiEchoServer) Count() int {
	mes.opLock.Get()
	defer mes.opLock.Release()
	return len(mes.clients)
}

// additional methods/functions

func (mes *multiEchoServer) Handle() {
	for {
		conn, err := mes.ln.Accept()
		if err != nil {
			continue
		}
		c := &client{cn: conn, wr: make(chan []byte, 100)}
		mes.addConn(c)
		go mes.write(c)
		go mes.read(c)
	}
}

func (mes *multiEchoServer) addConn(c *client) {
	mes.opLock.Get()
	defer mes.opLock.Release()
	mes.clients[c] = true
}

func (mes *multiEchoServer) killConn(t *client) {
	mes.opLock.Get()
	defer mes.opLock.Release()
	if err := t.cn.Close(); err != nil {
		fmt.Println(err)
	}
	delete(mes.clients, t)
}

func (mes *multiEchoServer) write(c *client) {
	for {
		select {
		case b := <-c.wr:
			if _, err := c.cn.Write(b); err != nil {
				fmt.Println("write err", err)
			}
		}
	}
}
func (mes *multiEchoServer) read(c *client) {
	reader := bufio.NewReader(c.cn)
	for {
		// Read up to and including the first '\n' character.
		msgBytes, err := reader.ReadBytes('\n')
		if err != nil {
			// 连接已被对端关闭
			if err == io.EOF {
				mes.killConn(c)
				return
			}
			fmt.Println("read err", err)
		}
		mes.BroadCast(msgBytes)
	}
}

func (mes *multiEchoServer) BroadCast(b []byte) {
	mes.opLock.Get()
	defer mes.opLock.Release()
	for client := range mes.clients {
		select {
		case client.wr <- b:
		default:
		}
	}
}

func NewConcurrencyResource(limit int) *ConcurrencyResource {
	r := &ConcurrencyResource{
		ch: make(chan struct{}, limit),
	}
	for i := 0; i < limit; i++ {
		r.ch <- struct{}{}
	}
	return r
}

type ConcurrencyResource struct {
	ch chan struct{}
}

func (r *ConcurrencyResource) Get() {
	<-r.ch
}

func (r *ConcurrencyResource) Release() {
	r.ch <- struct{}{}
}
