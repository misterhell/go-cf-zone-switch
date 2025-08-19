package servers

import "net"




type Address struct {
	addr string
}

func (a *Address) GetIP() *net.IPAddr {
	addr, _ := net.ResolveIPAddr("ip", a.addr)


	return addr
}


func GetServers() []Address {
	servers := []Address{
		{"176.9.70.13"},
		{"127.0.0.1"},

	}

	// TODO: parse addresses from toml config

	return servers
}
