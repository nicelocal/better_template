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
	c.Next()

	if !c.NextBlock() {
		return plugin.Error("better_template", c.SyntaxErr("{"))
	}
	domainMatcher := strmatcher.NewMphIndexMatcher()
	lookup := make(map[uint32]*entry)

	for c.Val() != "}" {
		m := c.Val()
		if !c.NextBlock() {
			return plugin.Error("better_template", c.SyntaxErr("{"))
		}
		e := &entry{make([]addressTtl, 0), make([]addressTtl, 0), ""}
		for {
			dst := c.Val()
			if dst == "}" {
				break
			}
			ip := net.ParseIP(dst)
			ttl := uint32(60)
			if c.NextLine() {
				tmp, err := strconv.Atoi(c.Val())
				if err != nil {
					return plugin.Error("better_template", c.ArgErr())
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
		if strings.HasPrefix(m, "regexp:") {
			t = strmatcher.Regex
			m = m[7:]
		} else if strings.HasPrefix(m, "keyword:") {
			t = strmatcher.Substr
			m = m[8:]
		} else if strings.HasPrefix(m, "subdomain:") {
			t = strmatcher.Domain
			m = m[10:]
			e.isSubdomainMatch = m
		}

		matcher, err := t.New(m)
		if err != nil {
			return plugin.Error("better_template", err)
		}

		lookup[domainMatcher.Add(matcher)] = e
	}

	domainMatcher.Build()

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return &BetterTemplate{Next: next, lookup: lookup, matcher: domainMatcher}
	})

	// All OK, return a nil error.
	return nil
}
