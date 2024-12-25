// Package example is a CoreDNS plugin that prints "example" to stdout on every packet received.
//
// It serves as an example CoreDNS plugin with numerous code comments.
package better_template

import (
	"context"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/v2fly/v2ray-core/v5/common/strmatcher"

	"github.com/miekg/dns"
)

// Define log to be a logger with the plugin name in it. This way we can just use log.Info and
// friends to log.
var log = clog.NewWithPlugin("better_template")

type addressTtl struct {
	ip  net.IP
	ttl uint32
}
type entry struct {
	ipv4  []addressTtl
	ipv6  []addressTtl
	fallT bool
}

type BetterTemplate struct {
	Next    plugin.Handler
	matcher *strmatcher.MphIndexMatcher
	lookup  map[uint32]*entry
}

// ServeDNS implements the plugin.Handler interface. This method gets called when example is used
// in a Server.
func (e *BetterTemplate) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	question := r.Question[0]
	matches := e.matcher.Match(strings.TrimSuffix(question.Name, "."))

	if len(matches) == 0 {
		return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
	}

	isV4 := question.Qtype == dns.TypeA
	if (!isV4 && question.Qtype != dns.TypeAAAA) || question.Qclass != dns.ClassINET {
		return dns.RcodeSuccess, nil
	}

	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	msg.RecursionAvailable = true

	for _, m := range matches {
		entry := e.lookup[m]
		if isV4 {
			for _, ip := range entry.ipv4 {
				msg.Answer = append(msg.Answer, &dns.A{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeA,
						Class:  question.Qclass,
						Ttl:    ip.ttl,
					},
					A: ip.ip,
				})
			}
		} else {
			for _, ip := range entry.ipv6 {
				msg.Answer = append(msg.Answer, &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeA,
						Class:  question.Qclass,
						Ttl:    ip.ttl,
					},
					AAAA: ip.ip,
				})
			}
		}
		if !entry.fallT {
			break
		}
	}

	w.WriteMsg(msg)

	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (e *BetterTemplate) Name() string { return "better_template" }
