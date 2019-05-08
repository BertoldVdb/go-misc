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
	Value []string
	TTL   uint32
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

// DNSMakeCurrentTime creates a responder that returns the current sever time
func DNSMakeCurrentTime(name string) func(query string) *DNSEntry {
	return func(query string) *DNSEntry {
		if query != "TXT,W|"+name {
			return nil
		}

		now := time.Now().UnixNano() * int64(time.Nanosecond)

		return &DNSEntry{TTL: 0, Value: []string{strconv.FormatInt(now, 10)}}
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
		for _, v := range value.Value {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: q.Qclass, Ttl: value.TTL},
				A:   net.ParseIP(v),
			})
		}
	}
}

func (h *DNSServer) handleAAAA(lcName string, msg *dns.Msg, q *dns.Question) {
	if value := h.doLookup("AAAA", lcName); value != nil {
		for _, v := range value.Value {
			msg.Answer = append(msg.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: q.Qclass, Ttl: value.TTL},
				AAAA: net.ParseIP(v),
			})
		}
	}
}

func (h *DNSServer) handleMX(lcName string, msg *dns.Msg, q *dns.Question) {
	if value := h.doLookup("MX", lcName); value != nil {
		for _, v := range value.Value {
			msg.Answer = append(msg.Answer, &dns.MX{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeMX, Class: q.Qclass, Ttl: value.TTL},
				Mx:  v,
			})
		}
	}
}

func (h *DNSServer) handleTXT(lcName string, msg *dns.Msg, q *dns.Question) {
	if value := h.doLookup("TXT", lcName); value != nil {
		if len(value.Value) > 0 {
			msg.Answer = append(msg.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: q.Qclass, Ttl: value.TTL},
				Txt: value.Value,
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
	if soaValue != nil && len(soaValue.Value) > 0 {
		msg.Authoritative = true

		sendNs = 1
	}

	tString := "Unknown"

	switch q.Qtype {
	case dns.TypeANY:
		h.handleA(lcName, &msg, q)
		h.handleAAAA(lcName, &msg, q)
		h.handleMX(lcName, &msg, q)
		h.handleTXT(lcName, &msg, q)
		tString = "ANY"
	case dns.TypeA:
		h.handleA(lcName, &msg, q)
		tString = "A"
	case dns.TypeAAAA:
		h.handleAAAA(lcName, &msg, q)
		tString = "AAAA"
	case dns.TypeMX:
		h.handleMX(lcName, &msg, q)
		tString = "MX"
	case dns.TypeTXT:
		h.handleTXT(lcName, &msg, q)
		tString = "TXT"
	case dns.TypeNS:
		if msg.Authoritative {
			if soaValue.Value[0] == lcName {
				sendNs = 2
			}
		}
		tString = "NS"
	case dns.TypeSOA:
		if msg.Authoritative {
			if soaValue.Value[0] == lcName {
				msg.Answer = append(msg.Answer, &dns.SOA{
					Hdr:     dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: soaValue.TTL},
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
		tString = "SOA"
	}

	//Add authority section, unless we want to send it as answer
	if sendNs > 0 {
		for _, l := range h.NSNames {
			ns := &dns.NS{
				Hdr: dns.RR_Header{Name: soaValue.Value[0], Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: soaValue.TTL},
				Ns:  l,
			}
			if sendNs == 1 {
				msg.Ns = append(msg.Ns, ns)
			} else if sendNs == 2 {
				msg.Answer = append(msg.Answer, ns)
			}
		}
	}

	h.Logger("SimpleDNS: Serving %s->%s: %s %s->%+v", w.RemoteAddr(), w.LocalAddr(), tString, lcName, msg.Answer)

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
	}

	serverTCP := &dns.Server{Addr: addr, Net: "tcp", Handler: h}
	return serverTCP.ListenAndServe()
}
