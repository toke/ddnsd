# Simple RFC2136 update HTTP server

State: Proof of Concept

The purpose of this project is to allow some home routers to
publish their IP to a RFC2136 Nameserver. ddnsd serves a http
server and responds to get requests and creates a RFC2136 update
message to update a remote dns server.

Example for Fritzbox:
`http://example.com:8080/api/dns/?ip=<ipaddr>&hostname=<domain>&token=<pass>`

The given "hostname" or "domainname" sould be relative to the zone specified in
config file.

ddnsd.conf:
```
---

Listen: ":8080"
BasePath: "/api"
Nameserver: a.ns.example.com:53
TTL: 3600
Zone: ddns.example.com.
Token: test
Secret: <secret>
```

## Installation

`go install github.com/toke/ddnsd@latest`

## Usage:


`ddnsd --config=configfile.yml`


Example systemd service file
```
[Unit]
After=network-online.target

[Service]
User=ddnsd
Group=ddnsd
LimitNOFILE=1024
ExecStart=/usr/local/bin/ddnsd
KillMode=control-group
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Bugs: maybe many, almost no error handling, crude state
