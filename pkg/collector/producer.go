/*
Copyright 2024 Richard Kosegi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collector

import (
	"github.com/netsampler/goflow2/v2/decoders/netflowlegacy"
	"github.com/netsampler/goflow2/v2/decoders/sflow"
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

func (p *producerMetricAdapter) Produce(msg any, args *producer.ProduceArgs) ([]producer.ProducerMessage, error) {
	tr := uint64(args.TimeReceived.UnixNano())
	sa, _ := args.SamplerAddress.Unmap().MarshalBinary()
	if rpt, ok := msg.(*netflowlegacy.PacketNetFlowV5); ok {
		msgs, err := protoproducer.ProcessMessageNetFlowLegacy(rpt)
		for _, x := range msgs {
			fmsg, ok := x.(*protoproducer.ProtoProducerMessage)
			if !ok {
				continue
			}
			fmsg.TimeReceivedNs = tr
			fmsg.SamplerAddress = sa
		}
		return msgs, err
	}
	if pkt, ok := msg.(*sflow.Packet); ok {
		return protoproducer.ProcessMessageSFlowConfig(pkt, nil)
	}
	return []producer.ProducerMessage{}, nil
}

func (p *producerMetricAdapter) Commit(messages []producer.ProducerMessage) {
	for _, msg := range messages {
		p.consumer.Consume(&(msg.(*protoproducer.ProtoProducerMessage)).FlowMessage)
	}
}

func (p *producerMetricAdapter) Close() {}
