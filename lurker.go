package lurker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/portmapping/lurker/nat"
)

const maxByteSize = 65520

// ListenResponse ...
type ListenResponse struct {
	Status int
	Addr   Addr
	Error  error
}

// Lurker ...
type Lurker interface {
	Listen() (c <-chan Connector, err error)
	RegisterListener(name string, listener Listener)
	Listener(name string) (Listener, bool)
	NetworkNAT(name string) nat.NAT
	Config() Config
}

type lurker struct {
	listeners  map[string]Listener
	cfg        *Config
	nat        nat.NAT
	sources    chan Source
	timeout    time.Duration
	connectors chan Connector
}

// NetworkNAT ...
func (l *lurker) NetworkNAT(name string) nat.NAT {
	listener, b := l.listeners[name]
	if b {
		ter, b := listener.(NATer)
		if b && ter.IsSupport() {
			return ter.NAT()
		}
	}
	return nil
}

// Listener ...
func (l *lurker) Listener(name string) (lis Listener, b bool) {
	lis, b = l.listeners[name]
	return
}

// PortUDP ...
func (l *lurker) Config() Config {
	return *l.cfg
}

// NAT ...
func (l *lurker) NAT() nat.NAT {
	return l.nat
}

// Stop ...
func (l *lurker) Stop() error {
	if err := l.nat.StopMapping(); err != nil {
		return err
	}
	fmt.Println("stopped")
	return nil
}

// New ...
func New(cfg *Config) Lurker {
	o := &lurker{
		cfg:        cfg,
		sources:    make(chan Source, 5),
		listeners:  make(map[string]Listener),
		connectors: make(chan Connector),
		timeout:    DefaultTimeout,
	}
	return o
}

// RegisterListener ...
func (l *lurker) RegisterListener(name string, listener Listener) {
	if name == "" {
		name = UUID()
	}
	l.listeners[name] = listener
}

func (l *lurker) waitingForReady() {
	total := len(l.listeners)
	for {
		count := 0
		for _, listener := range l.listeners {
			if listener.IsReady() {
				count++
			}
		}
		if count == total {
			return
		}
		time.Sleep(3 * time.Second)
	}
}

// Listen ...
func (l *lurker) Listen() (c <-chan Connector, err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Errorw("listener error found", "error", e)
		}
	}()

	var lis []string
	for name, listener := range l.listeners {
		lis = append(lis, name)
		go listener.Listen(l.connectors)

		if v, b := listener.(MappingListener); b {
			err := v.NAT().Mapping()
			if err != nil {
				return nil, err
			}
		}

	}
	l.waitingForReady()

	return l.connectors, nil
}

// JSON ...
func (r ListenResponse) JSON() []byte {
	marshal, err := json.Marshal(r)
	if err != nil {
		return nil
	}
	return marshal
}
