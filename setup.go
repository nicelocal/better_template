package better_template

import (
	"net"
	"strconv"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
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
		if c.Next() && c.Val() != "{" {
			return plugin.Error("better_template", c.SyntaxErr("{"))
		}
		if !c.Next() {
			return plugin.Error("better_template", c.ArgErr())
		}
		e := &entry{make([]addressTtl, 0), make([]addressTtl, 0), ""}
		for {
			dst := c.Val()
			if dst == "}" {
				c.NextLine()
				break
			}
			ip := net.ParseIP(dst)
			if ip == nil {
				return plugin.Error("better_template", c.ArgErr())
			}
			ttl := uint32(60)
			if c.NextArg() {
				tmp, err := strconv.Atoi(c.Val())
				if err != nil {
					return plugin.Error("better_template", err)
				}
				if tmp > 2147483647 || ttl < 0 {
					return plugin.Error("better_template", c.Err("Invalid TTL"))
				}
				ttl = uint32(tmp)
			}
			if temp := ip.To4(); temp != nil {
				e.ipv4 = append(e.ipv4, addressTtl{temp, ttl})
			} else {
				e.ipv6 = append(e.ipv6, addressTtl{ip, ttl})
			}
			if !c.NextLine() {
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
	}

	domainMatcher.Build()

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return &BetterTemplate{Next: next, lookup: lookup, matcher: domainMatcher}
	})

	// All OK, return a nil error.
	return nil
}
