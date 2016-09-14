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
//import "crypto/tls"
import "encoding/json"

/* command-line arguments */
var (
	port    = flag.Int("port", 32000, "listen on port number")
	dbglvl  = flag.Int("dbg", 1, "debugging level")
	workers = flag.Int("workers", 3, "number of independent workers")
	server  = flag.String("server", "https://dns.google.com", "server IP address")
)

/* logging stuff */
func dbg(lvl int, fmt string, v ...interface{}) { if (*dbglvl >= lvl) { dbglog.Printf(fmt, v...) } }
func die(msg error) { dbglog.Fatalln("fatal error:", msg.Error()) }
var dbglog = log.New(os.Stderr, "", log.LstdFlags | log.Lshortfile | log.LUTC)

/* global channels */
type Query struct { query *dns.Msg; rchan *chan Reply }
type Reply struct { query *dns.Msg; reply *dns.Msg }
var qchan = make(chan Query, 100)

/* UDP request handler */
func handle(buf []byte, addr *net.UDPAddr, uc *net.UDPConn) {
	dbg(2, "new request from %s (%d bytes)", addr, len(buf))

	/* try unpacking */
	msg := new(dns.Msg)
	err := msg.Unpack(buf)
	if (err != nil) { dbg(2, "msg.Unpack failed: %s", err); return }
	dbg(7, "unpacked: %s", msg)

	/* for each question */
	if (len(msg.Question) < 1) { dbg(2, "no questions"); return }
	for i,q := range msg.Question {
		dbg(3, "  [%d] type=%d class=%d name=%s", i, q.Qtype, q.Qclass, q.Name)
	}

	/* TODO: check cache */

	/* pass to resolvers and block until the response comes */
	rchan := make(chan Reply, 1)
	qchan <- Query{msg, &rchan}
	rs := <-rchan

	/* TODO: check for empty answers */

	/* TODO: check for packing errors? */
	rbuf,_ := rs.reply.Pack()
	uc.WriteToUDP(rbuf, addr)
}

/* resolves queries */
func resolver() {
	/* the HTTP client */
	// FIXME: proper TLS
//	var httpTr = &http.Transport{ TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	var httpTr = http.DefaultTransport.(*http.Transport)
	httpTr.ExpectContinueTimeout = 0 // fix for Go 1.6
	var httpClient = &http.Client{ Timeout: time.Second*10, Transport: httpTr }

	for {
		/* read from the query chan */
		qs := <-qchan
		qname := qs.query.Question[0].Name
		qtype := fmt.Sprintf("%d", qs.query.Question[0].Qtype)

		/* start the reply */
		reply := qs.query.Copy()
		reply.Compress = true

		/* prepare the query */
		v := url.Values{}
		v.Set("name", qname)
		v.Set("type", qtype)
		// TODO: random padding?
		addr := "https://http2.golang.org/reqinfo"
//		addr := fmt.Sprintf("%s/resolve?%s", *server, v.Encode())

		/* send the query */
		hreq,_ := http.NewRequest("GET", addr, nil)
//		hreq.Host = "dns.google.com"
		resp,err := httpClient.Do(hreq)
		if (err == nil && resp.StatusCode == 200) {
			/* read & parse JSON */
			buf,_ := ioutil.ReadAll(resp.Body)
			dbg(1, "%s", string(buf))
			resp.Body.Close()
			var j map[string]interface{}
			json.Unmarshal(buf, &j)

			dbg(3, "[%s/%s] %s %s: %+v", qname, qtype, resp.Status, resp.Proto, j)

			status,_  := j["Status"].(float64)
			if (status != 0) { dbg(1, "FIXME!") }

			for _,ans := range j["Answer"].([]interface{}) {
				//answers,_ := j["Answer"]
				dbg(3, "  %+v", ans)
			}

		} else if (err == nil) {
			dbg(1, "[%s/%s] invalid status: %s", qname, qtype, resp.Status)
			resp.Body.Close()
		} else {
			dbg(1, "[%s/%s] error: %s", qname, qtype, err.Error())
		}

		/* write the reply */
		*qs.rchan <- Reply{ qs.query, reply }
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
	for i := 0; i < *workers; i++ { go resolver() }

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
