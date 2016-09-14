/**
 * dingo: a DNS cache server in Go
 *
 * Copyright (C) 2016 Pawel Foremski <pawel@foremski.pl>
 */

package main

import "fmt"
import "os"
import "net"
import "flag"
import "log"
import "github.com/miekg/dns"
import "net/http"
import "net/url"
import "time"
import "io/ioutil"
import "encoding/json"
import "crypto/tls"

/**********************************************************************/

/* command-line arguments */
var (
	port    = flag.Int("port", 32000, "listen on port number")
	dbglvl  = flag.Int("dbg", 1, "debugging level")
	workers = flag.Int("workers", 3, "number of independent workers")
	server  = flag.String("server", "https://dns.google.com", "server IP address")
)

/**********************************************************************/

/* logging stuff */
func dbg(lvl int, fmt string, v ...interface{}) { if (*dbglvl >= lvl) { dbglog.Printf(fmt, v...) } }
func die(msg error) { dbglog.Fatalln("fatal error:", msg.Error()) }
var dbglog = log.New(os.Stderr, "", log.LstdFlags | log.Lshortfile | log.LUTC)

/* structures */
type GRR struct {
	Name   string
	Type   uint16
	TTL    uint32
	Data   string
}
type Reply struct {
	Status int
	TC     bool
	RD     bool
	RA     bool
	AD     bool
	CD     bool
	Question   []GRR
	Answer     []GRR
	Additional []GRR
	Authority  []GRR
	Comment    string
}

/* global channels */
type Query struct { Name string; Type int; rchan *chan Reply }
var qchan = make(chan Query, 100)

/**********************************************************************/

/* UDP request handler */
func handle(buf []byte, addr *net.UDPAddr, uc *net.UDPConn) {
	dbg(3, "new request from %s (%d bytes)", addr, len(buf))

	/* try unpacking */
	msg := new(dns.Msg)
	if err := msg.Unpack(buf); err != nil { dbg(3, "Unpack failed: %s", err); return }
	dbg(7, "unpacked: %s", msg)

	/* for each question */
	if (len(msg.Question) < 1) { dbg(3, "no questions"); return }

	/* TODO: check cache */

	/* pass to resolvers and block until the response comes */
	r := resolve(msg.Question[0].Name, int(msg.Question[0].Qtype))
	dbg(8, "got reply: %+v", r)

	/* rewrite the answers in r */
	rmsg := new(dns.Msg)
	rmsg.SetReply(msg)
	rmsg.Compress = true
	if (r.Status >= 0) {
		rmsg.Rcode = r.Status
		rmsg.Truncated = r.TC
		rmsg.RecursionDesired = r.RD
		rmsg.RecursionAvailable = r.RA
		rmsg.AuthenticatedData = r.AD
		rmsg.CheckingDisabled = r.CD

		for _,grr := range r.Answer { rmsg.Answer = append(rmsg.Answer, getrr(grr)) }
		for _,grr := range r.Authority { rmsg.Ns = append(rmsg.Ns, getrr(grr)) }
		for _,grr := range r.Additional { rmsg.Extra = append(rmsg.Extra, getrr(grr)) }
	} else {
		rmsg.Rcode = 2 // SERVFAIL
	}

	dbg(8, "sending %s", rmsg.String())
//	rmsg.Truncated = true

	/* pack and send! */
	rbuf,err := rmsg.Pack()
	if (err != nil) { dbg(2, "Pack() failed: %s", err); return }
	uc.WriteToUDP(rbuf, addr)
}

/* convert Google RR to miekg/dns RR */
func getrr(grr GRR) dns.RR {
	hdr := dns.RR_Header{Name: grr.Name, Rrtype: grr.Type, Class: dns.ClassINET, Ttl: grr.TTL }
	str := hdr.String() + grr.Data
	rr,err := dns.NewRR(str)
	if (err != nil) { dbg(3, "getrr(%s): %s", str, err.Error()) }
	return rr
}

/* pass to the request queue and wait until reply */
func resolve(name string, qtype int) Reply {
	rchan := make(chan Reply, 1)
	qchan <- Query{name, qtype, &rchan}
	return <-rchan
}

/* resolves queries */
func resolver(server string) {
	/* setup the HTTP client */
	var tlsCfg = &tls.Config{ ServerName: "dns.google.com" }
	var httpTr = http.DefaultTransport.(*http.Transport)
	httpTr.TLSClientConfig = tlsCfg;
//	req,_ := http.NewRequest("GET", "https://www.google.com/", nil)
//	httpTr.RoundTrip(req)
	var httpClient = &http.Client{ Timeout: time.Second*10, Transport: httpTr }

	for q := range qchan {
		/* make the new response object */
		r := Reply{ Status: -1 }

		/* prepare the query */
		v := url.Values{}
		v.Set("name", q.Name)
		v.Set("type", fmt.Sprintf("%d", q.Type))
		// TODO: random padding?

		addr     := fmt.Sprintf("%s/resolve?%s", server, v.Encode())
		hreq,_   := http.NewRequest("GET", addr, nil)
		hreq.Host = "dns.google.com"

		/* send the query */
		resp,err := httpClient.Do(hreq)
		if (err == nil) {
			dbg(2, "[%s/%d] %s %s", q.Name, q.Type, resp.Status, resp.Proto)

			/* read */
			buf,_ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			dbg(7, "  reply: %s", buf)

			/* parse JSON? */
			if (resp.StatusCode == 200) { json.Unmarshal(buf, &r) }
		} else { dbg(1, "[%s/%d] error: %s", q.Name, q.Type, err.Error()) }

		/* write the reply */
		*q.rchan <- r
	}
}

/* main */
func main() {
	/* prepare */
	flag.Parse()
	dbglog = log.New(os.Stderr, "", log.LstdFlags | log.Lshortfile | log.LUTC)

	/* listen */
	laddr   := net.UDPAddr{ Port: *port }
	uc, err := net.ListenUDP("udp", &laddr)
	if err != nil { die(err) }

	/* start workers */
	for i := 0; i < *workers; i++ { go resolver(*server) }

	/* accept new connections forever */
	dbg(1, "dingo ver. 0.1 started on UDP port %d", laddr.Port)
	var buf []byte
	for {
		buf = make([]byte, 1500)
		n, addr, err := uc.ReadFromUDP(buf)
		if err == nil { go handle(buf[0:n], addr, uc) }
	}

	uc.Close()
}
