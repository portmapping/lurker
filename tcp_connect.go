package lurker

import (
	"net"
	"time"

	"github.com/portmapping/lurker/common"
)

var _ Connector = &tcpConnector{}

type tcpConnector struct {
	id      func(id string)
	addr    func(addr common.Addr)
	timeout time.Duration
	conn    net.Conn
	ticker  *time.Ticker
}

// ConnectorListener ...
func (c *tcpConnector) ConnectorListener() ConnectorListener {
	return c
}

// Addr ...
func (c *tcpConnector) Addr(f func(addr common.Addr)) {
	c.addr = f
}

// Header ...
func (c *tcpConnector) Header() (HandshakeHead, error) {
	b := make([]byte, 8)
	_, err := c.conn.Read(b)
	if err != nil {
		return HandshakeHead{}, err
	}
	return ParseHandshakeByte(b)
}

// Reply ...
func (c *tcpConnector) Reply(status HandshakeStatus, data []byte) error {
	r := HandshakeResponse{
		Status: status,
		Data:   data,
	}

	if c.timeout != 0 {
		err := c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		if err != nil {
			log.Debugw("debug|Reply|SetReadDeadline", "error", err)
			return err
		}
	}
	_, err := c.conn.Write(r.JSON())
	if err != nil {
		return err
	}
	return nil
}

// RegisterCallback ...
func (c *tcpConnector) RegisterCallback(cb ConnectorCallback) {

}

// ID ...
func (c *tcpConnector) ID(f func(string)) {
	c.id = f
}

func newTCPConnector(conn net.Conn) Connector {
	c := &tcpConnector{
		timeout: 5 * time.Second,
		conn:    conn,
		//connector: connector,
	}
	return c
}

func (c *tcpConnector) interaction() (err error) {
	log.Debugw("interaction call")
	data := make([]byte, maxByteSize)
	if c.timeout != 0 {
		err := c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		if err != nil {
			log.Debugw("debug|Reply|SetReadDeadline", "error", err)
			return err
		}
	}
	log.Info("read data")
	n, err := c.conn.Read(data)
	if err != nil {
		log.Debugw("debug|Reply|Read", "error", err)
		return err
	}

	var r HandshakeRequest
	service, err := DecodeHandshakeRequest(data[:n], &r)
	if err != nil {
		log.Debugw("debug|Reply|DecodeHandshakeRequest", "error", err)
		return err
	}

	if c.id != nil {
		c.id(service.ID)
	}
	netAddr := common.ParseNetAddr(c.conn.RemoteAddr())
	log.Debugw("debug|Reply|ParseNetAddr", "common", netAddr)
	if c.addr != nil {
		c.addr(*netAddr)
	}
	var resp HandshakeResponse
	resp.Status = HandshakeStatusSuccess
	resp.Data = []byte("Connected")
	if c.timeout != 0 {
		err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
		if err != nil {
			log.Debugw("debug|Reply|SetWriteDeadline", "error", err)
			return err
		}
	}
	log.Info("write data")
	_, err = c.conn.Write(resp.JSON())
	if err != nil {
		log.Debugw("debug|Reply|Write", "error", err)
		return err
	}
	return nil
}

func (c *tcpConnector) intermediary() error {
	return nil
}

func (c *tcpConnector) other(ht HandshakeType) error {
	switch ht {
	case HandshakeReverse:
		addr := c.conn.RemoteAddr()
		dial, err := net.Dial("tcp", addr.String())
		if err != nil {
			return err
		}
		defer dial.Close()
	}
	return nil
}

// KeepConnect ...
func (c *tcpConnector) KeepConnect() {
	c.ticker = time.NewTicker(time.Second * 30)
	for {
		select {
		case <-c.ticker.C:
			//todo
			return
		}
	}
}

// Pong ...
func (c *tcpConnector) pong() error {
	return c.Reply(HandshakeStatusSuccess, []byte("PONG"))
}

// Do ...
func (c *tcpConnector) Do(ht HandshakeType) error {
	switch ht {
	case HandshakeTypePing:
		return c.pong()
	case HandshakeTypeConnect:
		return c.interaction()
	case HandshakeTypeAdapter:
		return c.intermediary()
	}
	return c.other(ht)
}

// Close ...
func (c *tcpConnector) Close() error {
	return c.conn.Close()
}
