package lurker

import (
	"context"
	"fmt"
	"net"

	"github.com/portmapping/lurker/common"
	"github.com/portmapping/lurker/nat"
)

type udpListener struct {
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	mappingPort int
	nat         nat.NAT
	listener    net.Listener
	udpListener *net.UDPConn
	cfg         *Config
	ready       bool
}

// IsReady ...
func (l *udpListener) IsReady() bool {
	return l.ready
}

// MappingPort ...
func (l *udpListener) MappingPort() int {
	return l.mappingPort
}

// Stop ...
func (l *udpListener) Stop() error {
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	return nil
}

// NewUDPListener ...
func NewUDPListener(cfg *Config) Listener {
	udp := &udpListener{
		ctx:    nil,
		cancel: nil,
		cfg:    cfg,
		port:   cfg.UDP,
	}
	udp.ctx, udp.cancel = context.WithCancel(context.TODO())
	return udp
}

// Listen ...
func (l *udpListener) Listen(c chan<- Connector) (err error) {
	udpAddr := common.LocalUDPAddr(l.port)
	//l.listener, err = kcp.Listen(udpAddr.String())
	//if err != nil {
	//	return err
	//}
	l.udpListener, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	fmt.Println("listen udp on common:", udpAddr.String())
	go listenUDP(l.ctx, l.udpListener, c)

	if !l.cfg.NAT {
		return nil
	}
	l.nat, err = Mapping("udp", l.cfg.UDP)
	if err != nil {
		return err
	}
	l.ready = true
	return nil
}

type udpHandshake struct {
	conn     *net.UDPConn
	addr     *net.UDPAddr
	connBack func(f Connector)
}

// Intermediary ...
func (h *udpHandshake) Intermediary() error {
	panic("implement me")
}

// Interaction ...
func (h *udpHandshake) Interaction() error {
	panic("implement me")
}

// Other ...
func (h *udpHandshake) Other() error {
	panic("implement me")
}

// Pong ...
func (h *udpHandshake) Pong() error {
	response := HandshakeResponse{
		Status: HandshakeStatusSuccess,
		Data:   []byte("PONG"),
	}
	write, err := h.conn.WriteToUDP(response.JSON(), h.addr)
	if err != nil {
		return err
	}
	if write == 0 {
		log.Warnw("write pong", "written", 0)
	}
	return nil
}

// Reply ...
func (h *udpHandshake) Reply() error {
	data := make([]byte, maxByteSize)
	n, addr, err := h.conn.ReadFromUDP(data)
	if err != nil {
		log.Debugw("debug|getClientFromTCP|Read", "error", err)
		return err
	}
	//ip, port := ParseAddr(common.String())
	//var r HandshakeRequest
	//service, err := DecodeHandshakeRequest(data[:n], &r)
	//if err != nil {
	//	log.Debugw("debug|getClientFromTCP|ParseService", "error", err)
	//	return err
	//}

	//c := source{
	//	common: Addr{
	//		Protocol: h.common.Network(),
	//		IP:       ip,
	//		Port:     port,
	//	},
	//	service: service,
	//}
	//h.connBack(&c)
	//todo:udpConnector
	_ = n

	netAddr := common.ParseNetAddr(h.addr)
	log.Debugw("debug|getClientFromTCP|ParseNetAddr", netAddr)
	var resp HandshakeResponse
	resp.Status = HandshakeStatusSuccess
	resp.Data = []byte("Connected")
	_, err = h.conn.WriteToUDP(resp.JSON(), addr)
	if err != nil {
		log.Debugw("debug|getClientFromTCP|write", "error", err)
		return err
	}
	return nil
}

// ConnectCallback ...
func (h *udpHandshake) ConnectCallback(f func(f Connector)) {
	h.connBack = f
}

// Do ...
func (h *udpHandshake) Do() (err error) {
	data := make([]byte, maxByteSize)
	var n int
	n, h.addr, err = h.conn.ReadFromUDP(data)
	if err != nil {
		//waiting for next
		log.Debugw("debug|listenUDP|ReadFromUDP", "error", err)
		return err
	}
	log.Debugw("received", "data", string(data[:n]))
	handshake, err := ParseHandshakeByte(data[:n])
	if err != nil {
		return err
	}
	return handshake.Run(h)
}

func listenUDP(ctx context.Context, listener *net.UDPConn, cli chan<- Connector) (err error) {

	for {
		select {
		case <-ctx.Done():
			return
		default:
			u := udpHandshake{
				conn: listener,
			}
			u.ConnectCallback(func(f Connector) {
				cli <- f
			})
			err = u.Do()
			if err != nil {
				continue
			}

			return nil
		}
	}
}
