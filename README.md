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
        # Fallthrough to the next match block is always disabled for multiple matching blocks
        # Priority: exact > subdomain > domain > regexp > keyword
        #
        # Fallthrough to the next plugin is always disabled if at least one block matches

        example.com IN A { # Exact match
            answer "{{ .Name }} 60 IN A 192.168.1.1"
        }
        example.com IN AAAA { # Exact match
            answer "{{ .Name }} 60 IN AAAA ff::123"
        }

        subdomain:example.com { # Subdomains only match (matches *.example.com)
            [...]
        }
        domain:example.com { # Subdomains or domain match (matches example.com and *.example.com)
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