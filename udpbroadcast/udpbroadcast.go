package udpbroadcast

import (
	"log"
	"net"
	"sync"
	"time"
)

type Client struct {
	sync.Mutex

	lastMessage time.Time
	addr        *net.UDPAddr

	timeoutInterval time.Duration
}

type UDPBroadcast struct {
	sync.RWMutex

	clients map[[18]byte]*Client
	socket  *net.UDPConn

	closeChan chan (struct{})

	TimeoutInterval      time.Duration
	ReceiveHandler       func(client interface{}, buf []byte)
	AllowDynamicNewPeers bool
}

func NewUDPBroadcast() (*UDPBroadcast, error) {
	u := &UDPBroadcast{}

	u.clients = make(map[[18]byte]*Client)
	u.TimeoutInterval = 30 * time.Second
	u.AllowDynamicNewPeers = true

	return u, nil
}

func (u *UDPBroadcast) timeoutHandler() {
	t := time.NewTicker(u.TimeoutInterval / 2)

	for {
		select {
		case <-u.closeChan:
			return
		case <-t.C:
		}

		u.Lock()
		for i, m := range u.clients {
			if m.timeoutInterval > 0 &&
				time.Since(m.lastMessage) > m.timeoutInterval {

				log.Println("Removed dynamic peer:", m.addr.String())

				delete(u.clients, i)
			}
		}
		u.Unlock()
	}
}

func addrToKey(addr *net.UDPAddr, key *[18]byte) {
	copy(key[:], addr.IP.To16())
	key[16] = byte(addr.Port)
	key[17] = byte(addr.Port >> 8)
}

func (u *UDPBroadcast) AddPeer(addr *net.UDPAddr, timeout time.Duration) *Client {
	client := &Client{
		addr:            addr,
		timeoutInterval: timeout,
	}

	var key [18]byte
	addrToKey(addr, &key)

	u.Lock()
	u.clients[key] = client
	u.Unlock()

	return client
}

func (u *UDPBroadcast) readHandler() {
	var lbuf [1600]byte

	for {
		n, addr, err := u.socket.ReadFromUDP(lbuf[:])
		if err != nil {
			return
		}
		buf := lbuf[:n]

		var key [18]byte
		addrToKey(addr, &key)

		u.RLock()
		client := u.clients[key]
		u.RUnlock()

		if client == nil {
			if u.AllowDynamicNewPeers {
				client = u.AddPeer(addr, u.TimeoutInterval)
				log.Println("Added dynamic peer:", addr.String())
			} else {
				log.Println("Received spurious packet from unknown peer:", addr.String())
				continue
			}
		}

		client.Lock()
		client.lastMessage = time.Now()
		client.Unlock()

		if u.ReceiveHandler != nil {
			u.ReceiveHandler(client, buf)
		}
	}
}

func (u *UDPBroadcast) ListenAndServe(addr string) error {
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	u.socket, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	u.closeChan = make(chan (struct{}))

	go u.timeoutHandler()
	u.readHandler()

	return nil
}

func (u *UDPBroadcast) Send(skip interface{}, buf []byte) error {
	u.RLock()
	for _, m := range u.clients {
		if m != skip {
			u.socket.WriteToUDP(buf, m.addr)
		}
	}
	u.RUnlock()

	return nil
}

func (u *UDPBroadcast) Shutdown() error {
	err := u.socket.Close()
	if err != nil {
		return err
	}

	close(u.closeChan)

	return nil
}
