// Package example is a CoreDNS plugin that prints "example" to stdout on every packet received.
//
// It serves as an example CoreDNS plugin with numerous code comments.
package better_template

import (
	"bytes"
	"context"
	"net"
	"strings"
	gotmpl "text/template"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/plugin/metrics"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
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
	qclass uint16
	qtype  uint16

	answer     []*gotmpl.Template
	additional []*gotmpl.Template
	authority  []*gotmpl.Template

	isSubdomainMatch string

	priority int
}

type templateData struct {
	Name     string
	Class    string
	Type     string
	Message  *dns.Msg
	Question *dns.Question
	Remote   string
	md       map[string]metadata.Func
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

	if question.Qclass != dns.ClassINET {
		return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
	}
	qName := strings.ToLower(strings.TrimSuffix(question.Name, "."))
	matches := e.matcher.Match(qName)
	if len(matches) == 0 {
		return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
	}

	var chosenEntry *entry
	chosenPriority := -1
	for _, m := range matches {
		entry := e.lookup[m]

		if entry.qclass != dns.ClassANY && question.Qclass != dns.ClassANY && question.Qclass != entry.qclass {
			continue
		}
		if entry.qtype != dns.TypeANY && question.Qtype != dns.TypeANY && question.Qtype != entry.qtype {
			continue
		}
		if qName == entry.isSubdomainMatch {
			continue
		}
		if entry.priority > chosenPriority {
			chosenEntry = entry
			chosenPriority = entry.priority
		}
	}

	if chosenEntry == nil {
		return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
	}

	state := request.Request{W: w, Req: r}
	data := &templateData{md: metadata.ValueFuncs(ctx), Remote: state.IP()}

	data.Name = state.Name()
	data.Question = &question
	data.Message = state.Req
	if question.Qclass != dns.ClassANY {
		data.Class = dns.ClassToString[question.Qclass]
	} else {
		data.Class = dns.ClassToString[chosenEntry.qclass]
	}
	if question.Qtype != dns.TypeANY {
		data.Type = dns.TypeToString[question.Qtype]
	} else {
		data.Type = dns.TypeToString[chosenEntry.qtype]
	}

	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	msg.RecursionAvailable = true

	for _, answer := range chosenEntry.answer {
		rr, err := executeRRTemplate(metrics.WithServer(ctx), metrics.WithView(ctx), "answer", answer, data)
		if err != nil {
			return dns.RcodeServerFailure, err
		}
		msg.Answer = append(msg.Answer, rr)
	}
	for _, additional := range chosenEntry.additional {
		rr, err := executeRRTemplate(metrics.WithServer(ctx), metrics.WithView(ctx), "additional", additional, data)
		if err != nil {
			return dns.RcodeServerFailure, err
		}
		msg.Extra = append(msg.Extra, rr)
	}
	for _, authority := range chosenEntry.authority {
		rr, err := executeRRTemplate(metrics.WithServer(ctx), metrics.WithView(ctx), "authority", authority, data)
		if err != nil {
			return dns.RcodeServerFailure, err
		}
		msg.Ns = append(msg.Ns, rr)
	}

	w.WriteMsg(msg)

	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (e *BetterTemplate) Name() string { return "better_template" }

func executeRRTemplate(server, view, section string, template *gotmpl.Template, data *templateData) (dns.RR, error) {
	buffer := &bytes.Buffer{}
	err := template.Execute(buffer, data)
	if err != nil {
		return nil, err
	}
	rr, err := dns.NewRR(buffer.String())
	if err != nil {
		return rr, err
	}
	return rr, nil
}
