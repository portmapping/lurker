package lurker

import (
	"context"
	"fmt"
	"github.com/portmapping/go-reuse"
	"net"

	p2pnat "github.com/libp2p/go-nat"
	"github.com/portmapping/lurker/nat"
)

type tcpListener struct {
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	mappingPort int
	nat         nat.NAT
	tcpListener net.Listener
	cfg         *Config
}

// NewTCPListener ...
func NewTCPListener(cfg *Config) Listener {
	tcp := &tcpListener{
		ctx:    nil,
		cancel: nil,
		port:   cfg.TCP,
		cfg:    cfg,
	}
	tcp.ctx, tcp.cancel = context.WithCancel(context.TODO())
	return tcp
}

// Listen ...
func (l *tcpListener) Listen(c chan<- Source) (err error) {
	tcpAddr := LocalTCPAddr(l.port)
	if l.cfg.Secret != nil {
		l.tcpListener, err = reuse.ListenTLS("tcp", DefaultLocalTCPAddr.String(), l.cfg.Secret)
	} else {
		l.tcpListener, err = reuse.ListenTCP("tcp", tcpAddr)
	}
	if err != nil {
		return err
	}
	go listenTCP(l.ctx, l.tcpListener, c)

	if !l.cfg.NAT {
		return nil
	}

	l.nat, err = nat.FromLocal("tcp", l.cfg.TCP)
	if err != nil {
		log.Debugw("nat error", "error", err)
		if err == p2pnat.ErrNoNATFound {
			fmt.Println("listen tcp on address:", tcpAddr.String())
		}
		l.cfg.NAT = false
	} else {
		extPort, err := l.nat.Mapping()
		if err != nil {
			log.Debugw("nat mapping error", "error", err)
			l.cfg.NAT = false
			return nil
		}
		l.mappingPort = extPort

		address, err := l.nat.GetExternalAddress()
		if err != nil {
			log.Debugw("get external address error", "error", err)
			l.cfg.NAT = false
			return nil
		}
		addr := ParseSourceAddr("tcp", address, extPort)
		fmt.Println("mapping on address:", addr.String())
	}
	return
}

// Stop ...
func (l *tcpListener) Stop() error {
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	return nil
}

func (l *tcpListener) listenTCP(ctx context.Context, listener net.Listener, cli chan<- Source) (err error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			acceptTCP, err := listener.Accept()
			if err != nil {
				log.Debugw("debug|getClientFromTCP|Accept", "error", err)
				continue
			}
			go l.getClientFromTCP(ctx, acceptTCP, cli)
		}
	}
}

func (l *tcpListener) getClientFromTCP(ctx context.Context, conn net.Conn, cli chan<- Source) error {
	close := true
	defer func() {
		if close {
			conn.Close()
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	default:
		data := make([]byte, maxByteSize)
		n, err := conn.Read(data)
		if err != nil {
			log.Debugw("debug|getClientFromTCP|Read", "error", err)
			return err
		}
		handshake, err := ParseHandshake(data)
		if err != nil {
			return err
		}
		switch handshake.Type {
		case RequestTypePing:

		case RequestTypeConnect:
		case RequestTypeAdapter:
		default:
		}

		ip, port := ParseAddr(conn.RemoteAddr().String())
		service, err := DecodeHandshakeRequest(data[:n])
		if err != nil {
			log.Debugw("debug|getClientFromTCP|ParseService", "error", err)
			return err
		}
		if service.KeepConnect {
			close = false
		}
		c := source{
			addr: Addr{
				Protocol: conn.RemoteAddr().Network(),
				IP:       ip,
				Port:     port,
			},
			service: service,
		}
		cli <- &c
		netAddr := ParseNetAddr(conn.RemoteAddr())

		err = tryReverseTCP(&source{addr: *netAddr,
			service: Service{
				ID:          GlobalID,
				KeepConnect: false,
			}})
		status := 0
		if err != nil {
			status = -1
			log.Debugw("debug|getClientFromTCP|tryReverseTCP", "error", err)
		}

		r := &ListenResponse{
			Status: status,
			Addr:   *netAddr,
			Error:  err,
		}
		_, err = conn.Write(r.JSON())
		if err != nil {
			log.Debugw("debug|getClientFromTCP|write", "error", err)
			return err
		}
	}
	return nil
}
