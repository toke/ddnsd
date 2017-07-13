package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/miekg/dns"
)

type Config struct {
	Listen     string `yaml:"Listen"`
	BaseUrl    string `yaml:"BaseUrl"`
	Zone       string `yaml:"Zone"`
	Nameserver string `yaml:"Nameserver"`
	TTL        uint32 `yaml:"TTL"`
	Token      string `yaml:"Token"`
	Secret     string `yaml:"Secret"`
}

/*
Example Config

---
Listen: ":8080"
Nameserver: a.ns.example.com:53
TTL: 3600
Zone: home.example.com.
Token: test
Secret: BASE64SECRET
*/

type Error interface {
	error
	Status() int
}

// /dns/
// http://127.0.0.1:8080/dns/?ip=127.0.0.3;hostname=test&token=test
// Fritzbox http://127.0.0.1:8080/dns/?ip=<ipaddr>;hostname=<domain>&token=<pass>
// dyn.2mydns.com/dyn.asp?username=<username>&password=<pass>&hostname=<domain>&myip=<ipaddr>

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome, %!", r.URL.Path[1:])

}

func main() {
	path := flag.String("config", "/etc/ddnsd.conf", "path to config file, eg. /home/user/ddnsd.conf")
	flag.Parse()

	conf, err := ioutil.ReadFile(*path)
	if err != nil {
		log.Fatal(err)
	}

	var c Config
	err = yaml.Unmarshal(conf, &c)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Using nameserver %s\n", c.Nameserver)
	fmt.Printf("Listening on %s\n", c.Listen)

	dnsupdate := makednsupdate(c)

	http.HandleFunc(c.BaseUrl+"/dns/", dnsupdate)
	http.ListenAndServe(c.Listen, nil)
}

func makednsupdate(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var fqdnhost string

		ip := r.URL.Query().Get("ip")
		hostname := r.URL.Query().Get("hostname")
		token := r.URL.Query().Get("token")

		var validHostname = regexp.MustCompile(`^[a-z0-9.\-]{2,100}$`)

		if token != cfg.Token {
			http.Error(w, "Not Authorized", http.StatusForbidden)
			return
		}

		if hostname == "" || ip == "" {
			http.Error(w, "Mandatory fields not set", http.StatusBadRequest)
			return
		}

		if !validHostname.MatchString(hostname) {
			http.Error(w, "Invalid Hostname", http.StatusBadRequest)
			return
		}

		// Only allow suffixed hostnames
		if strings.HasSuffix(hostname, cfg.Zone) {
			fqdnhost = hostname
		} else {
			fqdnhost = hostname + "." + cfg.Zone
		}

		fmt.Printf("Request for %s -> %s\n", fqdnhost, ip)

		rr := new(dns.A)
		rr.Hdr = dns.RR_Header{Name: fqdnhost, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(cfg.TTL)}
		rr.A = net.ParseIP(ip)
		rrs := []dns.RR{rr}

		m := new(dns.Msg)
		m.SetUpdate(cfg.Zone)
		m.RemoveRRset(rrs)
		m.Insert(rrs)

		c := new(dns.Client)
		c.SingleInflight = true

		c.TsigSecret = map[string]string{"axfr.": cfg.Secret}

		reply, _, err := c.Exchange(m, cfg.Nameserver)
		if err != nil {
			fmt.Printf("Error %s\n", err)
			http.Error(w, err.Error(), http.StatusFailedDependency)
		}
		if reply != nil && reply.Rcode != dns.RcodeSuccess {
			fmt.Printf("Error %s\n", dns.RcodeToString[reply.Rcode])
			http.Error(w, "Something went wrong", 400)
			fmt.Fprint(w, fmt.Errorf("DNS update failed. Server replied: %s", dns.RcodeToString[reply.Rcode]))
			return
		}
		fmt.Println("OK")
		fmt.Fprint(w, "OK")
	}
}
