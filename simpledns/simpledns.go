package simpledns

import (
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DNSEntry describes the record that will be returned.
type DNSEntry struct {
	value []string
	ttl   uint32
}

// Logger is a function interface that can be used for logging
type Logger func(string, ...interface{})

// DNSServer contains the configuration of the server.
type DNSServer struct {
	LocalMap       map[string](*DNSEntry)
	LookupFuncs    []func(query string) *DNSEntry
	NSNames        []string
	SoaMBox        string
	SpoofRecursive bool
	Logger         Logger
}

// DnsMakeCurrentTime creates a responder that returns the current sever time
func DnsMakeCurrentTime(name string) func(query string) *DNSEntry {
	return func(query string) *DNSEntry {
		if query != "TXT,W|"+name {
			return nil
		}

		now := time.Now().UnixNano() * int64(time.Nanosecond)

		return &DNSEntry{ttl: 0, value: []string{strconv.FormatInt(now, 10)}}
	}
}

func (h *DNSServer) doLookup(qt string, domain string) *DNSEntry {
	find := func(query string) *DNSEntry {
		for _, f := range h.LookupFuncs {
			if value := f(query); value != nil {
				return value
			}
		}

		if value, ok := h.LocalMap[query]; ok {
			return value
		}

		return nil
	}

	// Try direct
	if value := find(qt + "|" + domain); value != nil {
		return value
	}

	//Try wildcard
	name := domain
	for {
		if value := find(qt + ",W|" + name); value != nil {
			return value
		}

		indexDot := strings.Index(name, ".")
		if indexDot < 0 {
			break
		}

		name = name[indexDot+1:]
	}

	return nil
}

func (h *DNSServer) handleA(lcName string, msg *dns.Msg, q *dns.Question) {
	if value := h.doLookup("A", lcName); value != nil {
		for _, v := range value.value {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: q.Qclass, Ttl: value.ttl},
				A:   net.ParseIP(v),
			})
		}
	}
}

func (h *DNSServer) handleAAAA(lcName string, msg *dns.Msg, q *dns.Question) {
	if value := h.doLookup("AAAA", lcName); value != nil {
		for _, v := range value.value {
			msg.Answer = append(msg.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: q.Qclass, Ttl: value.ttl},
				AAAA: net.ParseIP(v),
			})
		}
	}
}

func (h *DNSServer) handleMX(lcName string, msg *dns.Msg, q *dns.Question) {
	if value := h.doLookup("MX", lcName); value != nil {
		for _, v := range value.value {
			msg.Answer = append(msg.Answer, &dns.MX{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeMX, Class: q.Qclass, Ttl: value.ttl},
				Mx:  v,
			})
		}
	}
}

func (h *DNSServer) handleTXT(lcName string, msg *dns.Msg, q *dns.Question) {
	if value := h.doLookup("TXT", lcName); value != nil {
		if len(value.value) > 0 {
			msg.Answer = append(msg.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: q.Qclass, Ttl: value.ttl},
				Txt: value.value,
			})
		}
	}
}

// ServeDNS is the function that serves the DNS requests.
func (h *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = false
	msg.RecursionAvailable = h.SpoofRecursive

	q := &r.Question[0]
	lcName := strings.ToLower(q.Name)

	sendNs := 0

	soaValue := h.doLookup("SOA", lcName)
	if soaValue != nil && len(soaValue.value) > 0 {
		msg.Authoritative = true

		sendNs = 1
	}

	switch q.Qtype {
	case dns.TypeANY:
		h.handleA(lcName, &msg, q)
		h.handleAAAA(lcName, &msg, q)
		h.handleMX(lcName, &msg, q)
		h.handleTXT(lcName, &msg, q)
	case dns.TypeA:
		h.handleA(lcName, &msg, q)
	case dns.TypeAAAA:
		h.handleAAAA(lcName, &msg, q)
	case dns.TypeMX:
		h.handleMX(lcName, &msg, q)
	case dns.TypeTXT:
		h.handleTXT(lcName, &msg, q)
	case dns.TypeNS:
		if msg.Authoritative {
			if soaValue.value[0] == lcName {
				sendNs = 2
			}
		}
	case dns.TypeSOA:
		if msg.Authoritative {
			if soaValue.value[0] == lcName {
				msg.Answer = append(msg.Answer, &dns.SOA{
					Hdr:     dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: soaValue.ttl},
					Ns:      h.NSNames[0],
					Mbox:    h.SoaMBox,
					Serial:  uint32(time.Now().UnixNano() * int64(time.Nanosecond) / int64(time.Second)),
					Refresh: 21600,
					Retry:   3600,
					Expire:  3600 * 24,
					Minttl:  300,
				})
			}
		}
	}

	//Add authority section, unless we want to send it as answer
	if sendNs > 0 {
		for _, l := range h.NSNames {
			ns := &dns.NS{
				Hdr: dns.RR_Header{Name: soaValue.value[0], Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: soaValue.ttl},
				Ns:  l,
			}
			if sendNs == 1 {
				msg.Ns = append(msg.Ns, ns)
			} else if sendNs == 2 {
				msg.Answer = append(msg.Answer, ns)
			}
		}
	}

    h.Logger("SimpleDNS: Serving %s->%s: %+v", w.RemoteAddr(), w.LocalAddr(), msg)

	w.WriteMsg(&msg)
}

func emailToMBox(email string) string {
	parts := strings.SplitN(email, "@", 2)
	parts[0] = strings.Replace(parts[0], ".", "\\.", -1)

	return strings.Join(parts, ".") + "."
}

// NewDNSServer creates an empty DNSServer object.
func NewDNSServer(nsName []string, email string) *DNSServer {
	h := &DNSServer{}
	h.NSNames = nsName
	h.SoaMBox = emailToMBox(email)
	h.LocalMap = make(map[string](*DNSEntry))
	h.Logger = log.Printf
	return h
}

// ListenAndServe will handle queries
func (h *DNSServer) ListenAndServe(addr string, udp bool) error {
	if udp {
		serverUDP := &dns.Server{Addr: addr, Net: "udp", UDPSize: 4096, Handler: h}
		return serverUDP.ListenAndServe()
	} else {
		serverTCP := &dns.Server{Addr: addr, Net: "tcp", Handler: h}
		return serverTCP.ListenAndServe()
	}
}
