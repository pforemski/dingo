/**
 * dingo: a DNS caching proxy written in Go
 *
 * Copyright (C) 2016-2017 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 *
 * NOTE: this software is under development, far from being complete
 */

package main

import "fmt"
import "os"
import "net"
import "flag"
import "log"
import "github.com/miekg/dns"
import "time"
import "github.com/patrickmn/go-cache"
import "math/rand"

/**********************************************************************/

/* command-line arguments */
var (
	opt_bindip = flag.String("bind", "127.0.0.1", "IP address to bind to")
	opt_port   = flag.Int("port", 32000, "listen on port number")
	opt_h1     = flag.Bool("h1", false, "use HTTPS/1.1 transport")
	opt_quic   = flag.Bool("quic", false, "use experimental QUIC transport")
	opt_dbglvl = flag.Int("dbg", 2, "debugging level")
)

/**********************************************************************/

/* logging stuff */
func dbg(lvl int, fmt string, v ...interface{}) {
	if *opt_dbglvl >= lvl {
		dbglog.Printf(fmt, v...)
	}
}
func die(msg error) { dbglog.Fatalln("fatal error:", msg.Error()) }

var dbglog *log.Logger

/* structures */
type GRR struct {
	Name string
	Type uint16
	TTL  uint32
	Data string
}
type Reply struct {
	Status     int
	TC         bool
	RD         bool
	RA         bool
	AD         bool
	CD         bool
	Question   []GRR
	Answer     []GRR
	Additional []GRR
	Authority  []GRR
	Comment    string
	Now        time.Time
}

/* global channels */
type Query struct {
	Name  string
	Type  int
	rchan *chan Reply
}

var qchan = make(chan Query, 100)

/* global reply cache */
var rcache *cache.Cache

/* module interface */
var Modules = make(map[string]Module)

type Module interface {
	Init()
	Start()
}

func register(name string, mod Module) *Module {
	Modules[name] = mod
	return &mod
}

/**********************************************************************/

/* UDP request handler */
func handle(buf []byte, addr *net.UDPAddr, uc *net.UDPConn) {
	/* try unpacking */
	msg := new(dns.Msg)
	if err := msg.Unpack(buf); err != nil {
		dbg(3, "unpack failed: %s", err)
		return
	} else {
		dbg(7, "unpacked message: %s", msg)
	}

	/* any questions? */
	if len(msg.Question) < 1 {
		dbg(3, "no questions")
		return
	}

	qname := msg.Question[0].Name
	qtype := msg.Question[0].Qtype
	dbg(2, "resolving %s/%s", qname, dns.TypeToString[qtype])

	/* check cache */
	var r Reply
	cid := fmt.Sprintf("%s/%d", qname, qtype)
	if x, found := rcache.Get(cid); found {
		// FIXME: update TTLs
		r = x.(Reply)
	} else {
		/* pass to resolvers and block until the response comes */
		r = resolve(qname, int(qtype))
		dbg(8, "got reply: %+v", r)

		/* put to cache for 10 seconds (FIXME: use minimum TTL) */
		rcache.Set(cid, r, 10*time.Second)
	}

	/* rewrite the answers in r into rmsg */
	rmsg := new(dns.Msg)
	rmsg.SetReply(msg)
	rmsg.Compress = true
	if r.Status >= 0 {
		rmsg.Rcode = r.Status
		rmsg.Truncated = r.TC
		rmsg.RecursionDesired = r.RD
		rmsg.RecursionAvailable = r.RA
		rmsg.AuthenticatedData = r.AD
		rmsg.CheckingDisabled = r.CD

		for _, grr := range r.Answer {
			rmsg.Answer = append(rmsg.Answer, getrr(grr))
		}
		for _, grr := range r.Authority {
			rmsg.Ns = append(rmsg.Ns, getrr(grr))
		}
		for _, grr := range r.Additional {
			rmsg.Extra = append(rmsg.Extra, getrr(grr))
		}
	} else {
		rmsg.Rcode = 2 // SERVFAIL
	}

	dbg(8, "sending %s", rmsg.String())
	//	rmsg.Truncated = true

	/* pack and send! */
	rbuf, err := rmsg.Pack()
	if err != nil {
		dbg(2, "Pack() failed: %s", err)
		return
	}
	uc.WriteToUDP(rbuf, addr)
}

/* convert Google RR to miekg/dns RR */
func getrr(grr GRR) dns.RR {
	hdr := dns.RR_Header{Name: grr.Name, Rrtype: grr.Type, Class: dns.ClassINET, Ttl: grr.TTL}
	str := hdr.String() + grr.Data
	rr, err := dns.NewRR(str)
	if err != nil {
		dbg(3, "getrr(%s): %s", str, err.Error())
	}
	return rr
}

/* pass to the request queue and wait until reply */
func resolve(name string, qtype int) Reply {
	rchan := make(chan Reply, 1)
	qchan <- Query{name, qtype, &rchan}
	return <-rchan
}

/* main */
func main() {
	rand.Seed(time.Now().UnixNano())
	dbglog = log.New(os.Stderr, "", log.LstdFlags|log.LUTC)

	/* prepare */
	for _, mod := range Modules {
		mod.Init()
	}
	flag.Parse()
	rcache = cache.New(24*time.Hour, 60*time.Second)

	/* listen */
	laddr := net.UDPAddr{IP: net.ParseIP(*opt_bindip), Port: *opt_port}
	uc, err := net.ListenUDP("udp", &laddr)
	if err != nil {
		die(err)
	}

	/* start workers */
	for _, mod := range Modules {
		mod.Start()
	}

	/* accept new connections forever */
	dbg(1, "dingo ver. 0.13 listening on %s UDP port %d", *opt_bindip, laddr.Port)
	var buf []byte
	for {
		buf = make([]byte, 1500)
		n, addr, err := uc.ReadFromUDP(buf)
		if err == nil {
			go handle(buf[0:n], addr, uc)
		}
	}

	uc.Close()
}
