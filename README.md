# better_template

An improved and simplified version of the coredns template plugin.  


Usage:

Add the following to `plugins.cfg`, before `debug`:

```
better_template:github.com/nicelocal/better_template
```

And use the following example Corefile:
```
. {
    better_template {
        # Fallthrough is always enabled for multiple matching blocks

        example.com { # Exact match
            192.168.1.1 [ TTL ]
            ff::123 [ TTL ]
            [...]
        }
        subdomain:example.com { # Subdomains only match
            [...]
        }
        regexp:exampl?e.com { # Regex match
            [...]
        }
        keyword:le.com { # Keyword match (domain contains)
            [...]
        }
    }

    forward . 8.8.8.8
}
```

Example dockerfile:

```
FROM golang

RUN git clone -b v1.12.0 --depth 1 https://github.com/coredns/coredns /coredns && \
    cd /coredns && \
    sed '/bind:bind/a better_template:github.com/nicelocal/better_template' plugin.cfg -i && \
    make

FROM scratch

COPY --from=0 /coredns/coredns /coredns

ENTRYPOINT ["/coredns"]
```