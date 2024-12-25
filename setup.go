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
		recordsV4 := make([]addressTtl, 0)
		recordsV6 := make([]addressTtl, 0)
		fallT := false
		for {
			dst := c.Val()
			if dst == "}" {
				break
			}
			if dst == "fallthrough" {
				fallT = true
				continue
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
				recordsV4 = append(recordsV4, addressTtl{temp, ttl})
			} else {
				recordsV6 = append(recordsV6, addressTtl{ip, ttl})
			}
			if !c.NextLine() {
				return plugin.Error("better_template", c.ArgErr())
			}
		}

		t := strmatcher.Domain
		if strings.HasPrefix(m, "regexp:") {
			t = strmatcher.Regex
			m = m[7:]
		} else if strings.HasPrefix(m, "keyword:") {
			t = strmatcher.Substr
			m = m[8:]
		} else if strings.HasPrefix(m, "full:") {
			t = strmatcher.Full
			m = m[4:]
		} else if strings.HasPrefix(m, "domain:") {
			m = m[7:]
		}

		matcher, err := t.New(m)
		if err != nil {
			return plugin.Error("better_template", err)
		}

		lookup[domainMatcher.Add(matcher)] = &entry{recordsV4, recordsV6, fallT}
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return &BetterTemplate{Next: next, lookup: lookup, matcher: domainMatcher}
	})

	// All OK, return a nil error.
	return nil
}
