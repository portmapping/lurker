package main

import (
	"github.com/portmapping/lurker"
	"github.com/portmapping/lurker/common"
	"github.com/spf13/cobra"
	"net"
)

func cmdCheck() *cobra.Command {
	var addr string
	var local int
	var network string
	var proxy string
	var proxyPort int
	var proxyName string
	var proxyPass string
	var bindPort int
	var id string
	var test bool
	cmd := &cobra.Command{
		Use: "check",
		Run: func(cmd *cobra.Command, args []string) {
			addrs, i := common.ParseAddr(addr)
			localAddr := net.IPv4zero
			ispAddr := net.IPv4zero

			cfg := lurker.DefaultConfig()
			var err error
			mport := bindPort
			if !test && proxy != "" {
				cfg.Proxy = []lurker.Proxy{
					{
						Type: proxy,
						Nat:  true,
						Port: proxyPort,
						Name: proxyName,
						Pass: proxyPass,
					},
				}
				l := lurker.New(cfg)
				mport, err = lurker.RegisterLocalProxy(l, cfg)
				if err != nil {
					panic(err)
				}
				go l.ListenOnMonitor()
			}

			s := lurker.NewSource(lurker.Service{
				ID:    id,
				ISP:   ispAddr,
				Local: localAddr,
			}, common.Addr{
				Protocol: network,
				IP:       addrs,
				Port:     i,
			})
			if bindPort != 0 {
				mapping, err := lurker.Mapping("tcp", bindPort)
				if err != nil {
					panic(err)
				}
				mport = mapping.ExtPort()
			}
			s.SetMappingPort("tcp", mport)
			err = s.Connect()
			if err != nil {
				panic(err)
			}
			waitForSignal()
		},
	}
	cmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1:16004", "default 127.0.0.1:16004")
	cmd.Flags().StringVarP(&network, "network", "n", "tcp", "")
	cmd.Flags().IntVarP(&local, "local", "l", 16004, "handle local mapping port")
	cmd.Flags().StringVarP(&proxy, "proxy", "p", "socks5", "locak proxy")
	cmd.Flags().StringVarP(&proxyName, "pname", "", "", "local proxy port")
	cmd.Flags().StringVarP(&proxyPass, "ppass", "", "", "local proxy port")
	cmd.Flags().IntVarP(&proxyPort, "pport", "", 10080, "local proxy port")
	cmd.Flags().IntVarP(&bindPort, "bind", "b", 0, "set bind port")
	cmd.Flags().BoolVarP(&test, "test", "t", false, "set test flag")
	cmd.Flags().StringVarP(&id, "id", "", lurker.GlobalID, "set the connect id")
	return cmd
}
