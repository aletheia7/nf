// Copyright 2016 aletheia7. All rights reserved. Use of this source code is
// governed by a BSD-2-Clause license that can be found in the LICENSE file.

package main

//go:generate dash -c "cd vendor/libnetfilter_queue-1.0.3 && ./configure --enable-static=yes --enable-shared=no"
//go:generate dash -c "cd vendor/libnetfilter_queue-1.0.3 && make"
import (
	"bytes"
	"github.com/Telefonica/nfqueue"
	"github.com/aletheia7/gogroup"
	"github.com/aletheia7/sd"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Install automake-1.15 when errors in go generate complain about aclocal*
// Need libnetlink-dev, libmnl-dev

var (
	j  = sd.New(sd.Set_default_disable_journal(true), sd.Set_default_writer_stdout())
	gg = gogroup.New(gogroup.Add_signals(gogroup.Unix))
)

type Queue struct {
	id    uint16
	queue *nfqueue.Queue
}

func New_queue(id uint16) *Queue {
	q := &Queue{
		id: id,
	}
	q.queue = nfqueue.NewQueue(q.id, q,
		&nfqueue.QueueConfig{
			MaxPackets: 5000,
			BufferSize: 16 * 1024 * 1024,
		})
	return q
}

func (o *Queue) Start(gg *gogroup.Group) error {
	return o.queue.Start()
}

func (o *Queue) Stop() error {
	return o.queue.Stop()
}

func (o *Queue) Handle(p *nfqueue.Packet) {
	var ip4 layers.IPv4
	var tcp layers.TCP
	var udp layers.UDP
	var payload gopacket.Payload
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &ip4, &tcp, &udp, &payload)
	parser.IgnorePanic = true
	parser.IgnoreUnsupported = true
	decoded := make([]gopacket.LayerType, 0, 10)
	err := parser.DecodeLayers(p.Buffer, &decoded)
	if err != nil {
		j.Err("DecodeLayers err", err)
	}
	if tcp.DstPort == 9000 {
		if 0 < tcp.SrcPort {
			j.Infof("drop tcp: %v:%d -> %v:%d %s\n", ip4.SrcIP, tcp.SrcPort, ip4.DstIP,
				tcp.DstPort, bytes.TrimRight(payload, "\n"))
		}
		if 0 < udp.SrcPort {
			j.Infof("drop udp: %v:%d -> %v:%d %s\n", ip4.SrcIP, udp.SrcPort, ip4.DstIP,
				udp.DstPort, bytes.TrimRight(payload, "\n"))
		}
		if err := p.Drop(); err != nil {
			j.Err(err)
		}
	} else {
		if 0 < tcp.SrcPort {
			j.Infof("accept tcp: %v:%d -> %v:%d %s\n", ip4.SrcIP, tcp.SrcPort, ip4.DstIP,
				tcp.DstPort, bytes.TrimRight(payload, "\n"))
		}
		if 0 < udp.SrcPort {
			j.Infof("accpet udp: %v:%d -> %v:%d %s\n", ip4.SrcIP, udp.SrcPort, ip4.DstIP,
				udp.DstPort, bytes.TrimRight(payload, "\n"))
		}
		if err := p.Accept(); err != nil {
			j.Err(err)
		}
	}
}

func main() {
	q := New_queue(77)
	go q.Start(gg)
	defer gg.Wait()
	<-gg.Done()
	q.Stop()
}
