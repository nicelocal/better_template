package better_template

import (
	"strconv"
	"strings"

	gotmpl "text/template"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	"github.com/v2fly/v2ray-core/v5/common/strmatcher"
)

func init() { plugin.Register("better_template", setup) }

func setup(c *caddy.Controller) error {
	domainMatcher := strmatcher.NewMphIndexMatcher()
	lookup := make(map[uint32]*entry)

	c.Next()
	for c.Val() != "" {
		if c.Val() == "better_template" {
			if c.Next() && c.Val() != "{" {
				return plugin.Error("better_template", c.SyntaxErr("{"))
			}
			c.Next()
		}

		m := c.Val()

		if !c.NextArg() {
			return plugin.Error("better_template", c.ArgErr())
		}
		class, ok := dns.StringToClass[c.Val()]
		if !ok {
			return plugin.Error("better_template", c.Errf("invalid query class %s", c.Val()))
		}

		if !c.NextArg() {
			return plugin.Error("better_template", c.ArgErr())
		}
		qtype, ok := dns.StringToType[c.Val()]
		if !ok {
			return plugin.Error("better_template", c.Errf("invalid RR class %s", c.Val()))
		}

		e := &entry{class, qtype, make([]*gotmpl.Template, 0), make([]*gotmpl.Template, 0), make([]*gotmpl.Template, 0), "", 0}

		for c.NextBlock() {
			switch c.Val() {
			case "answer":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return plugin.Error("better_template", c.ArgErr())
				}
				for _, answer := range args {
					tmpl, err := newTemplate("answer", answer)
					if err != nil {
						return plugin.Error("better_template", c.Errf("could not compile template: %s, %v", c.Val(), err))
					}
					e.answer = append(e.answer, tmpl)
				}

			case "additional":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return plugin.Error("better_template", c.ArgErr())
				}
				for _, additional := range args {
					tmpl, err := newTemplate("additional", additional)
					if err != nil {
						return plugin.Error("better_template", c.Errf("could not compile template: %s, %v\n", c.Val(), err))
					}
					e.additional = append(e.additional, tmpl)
				}

			case "authority":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return plugin.Error("better_template", c.ArgErr())
				}
				for _, authority := range args {
					tmpl, err := newTemplate("authority", authority)
					if err != nil {
						return plugin.Error("better_template", c.Errf("could not compile template: %s, %v\n", c.Val(), err))
					}
					e.authority = append(e.authority, tmpl)
				}
			default:
				return plugin.Error("better_template", c.ArgErr())
			}
		}

		t := strmatcher.Full
		e.priority = 4
		if strings.HasPrefix(m, "subdomain:") {
			t = strmatcher.Domain
			m = m[10:]
			e.isSubdomainMatch = m
			e.priority = 3
		} else if strings.HasPrefix(m, "domain:") {
			t = strmatcher.Domain
			m = m[7:]
			e.priority = 2
		} else if strings.HasPrefix(m, "regexp:") {
			t = strmatcher.Regex
			m = m[7:]
			e.priority = 1
		} else if strings.HasPrefix(m, "keyword:") {
			t = strmatcher.Substr
			m = m[8:]
			e.priority = 0
		}

		matcher, err := t.New(m)
		if err != nil {
			return plugin.Error("better_template", err)
		}

		lookup[domainMatcher.Add(matcher)] = e

		if c.Val() == "}" && !c.Next() {
			break
		}
		if c.Val() == "}" {
			break
		}
	}

	domainMatcher.Build()

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return &BetterTemplate{Next: next, lookup: lookup, matcher: domainMatcher}
	})

	// All OK, return a nil error.
	return nil
}

func newTemplate(name, text string) (*gotmpl.Template, error) {
	funcMap := gotmpl.FuncMap{
		"parseInt": strconv.ParseUint,
	}
	return gotmpl.New(name).Funcs(funcMap).Parse(text)
}
