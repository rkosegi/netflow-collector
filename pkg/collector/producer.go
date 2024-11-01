package collector

import (
	"github.com/netsampler/goflow2/v2/decoders/netflowlegacy"
	flowpb "github.com/netsampler/goflow2/v2/pb"
	"github.com/netsampler/goflow2/v2/producer"
	protoproducer "github.com/netsampler/goflow2/v2/producer/proto"
)

type messageConsumer interface {
	Consume(msg *flowpb.FlowMessage)
}

type producerMetricAdapter struct {
	consumer messageConsumer
}

func (p *producerMetricAdapter) Produce(msg interface{}, args *producer.ProduceArgs) ([]producer.ProducerMessage, error) {
	tr := uint64(args.TimeReceived.UnixNano())
	sa, _ := args.SamplerAddress.Unmap().MarshalBinary()
	if rpt, ok := msg.(*netflowlegacy.PacketNetFlowV5); ok {
		rpt, err := protoproducer.ProcessMessageNetFlowLegacy(rpt)
		for _, x := range rpt {
			fmsg, ok := x.(*protoproducer.ProtoProducerMessage)
			if !ok {
				continue
			}
			fmsg.TimeReceivedNs = tr
			fmsg.SamplerAddress = sa
		}
		return rpt, err
	}
	return []producer.ProducerMessage{}, nil
}

func (p *producerMetricAdapter) Commit(messages []producer.ProducerMessage) {
	for _, msg := range messages {
		p.consumer.Consume(&(msg.(*protoproducer.ProtoProducerMessage)).FlowMessage)
	}
}

func (p *producerMetricAdapter) Close() {}
