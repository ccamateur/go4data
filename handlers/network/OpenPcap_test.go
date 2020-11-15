package network

import (
	"context"
	"testing"

	"github.com/percybolmer/workflow/metric"
	"github.com/percybolmer/workflow/payload"
	"github.com/percybolmer/workflow/pubsub"
)

func TestOpenPcapHandle(t *testing.T) {
	pcapHandler := NewOpenPcapHandler()

	testPayload := payload.BasePayload{
		Source:  "test",
		Payload: []byte(`testing/tcpSpoof.pcap`),
	}
	pcapHandler.SetMetricProvider(metric.NewPrometheusProvider(), "pcaphandler")
	pcapHandler.bpf = "tcp and ip"
	err := pcapHandler.Handle(context.Background(), testPayload, "packets")
	if err != nil {
		t.Fatal(err)
	}

	pipe, err := pubsub.Subscribe("packets", 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	for pay := range pipe.Flow {
		netpay, err := NewPayload(pay)
		if err != nil {
			t.Fatal(err)
		}
		if len(netpay.Payload.Data()) == 0 {
			t.Fatalf("Wrong packet length, %s", netpay.Payload.Dump())
		}
		t.Log(netpay.Payload.Dump())
	}
}
