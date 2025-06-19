package storage

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/kokaq/core/queue"
	"github.com/kokaq/protocol/tcp"
)

type Heap struct {
	channel    chan ChannelInput
	actualHeap *queue.Kokaq
}

type ChannelInput struct {
	request         *tcp.Request
	responseChannel chan *tcp.Response
}

func NewHeap(channel chan ChannelInput, namespaceId uint32, queueId uint32) *Heap {
	actualHeap, err := queue.NewDefaultKokaq(namespaceId, queueId)

	if err != nil {
		panic("Error creating heap")
	} else {
		return &Heap{
			channel:    channel,
			actualHeap: actualHeap,
		}
	}
}

func (hp *Heap) handle() {
	// keep receiving requests from heap channel
	for {
		select {
		case rcv, more := <-hp.channel:
			// TODO: Add validations like this to all channels
			if !more || rcv.request == nil || rcv.responseChannel == nil {
				return
			}
			res := rcv.request.ToResponse()
			switch rcv.request.GetOpcode() {
			case 0x1: // Create
				res.SetStatus(tcp.ResponseStatusSuccess)
				res.SetPayload(make([]byte, 0))
				break
			case 0x2: // Delete
				res.SetStatus(tcp.ResponseStatusSuccess)
				res.SetPayload(make([]byte, 0))
				break
			case 0x4: // Peek
				qi, err := hp.actualHeap.PeekItem()
				if err != nil {
					res.SetStatus(tcp.ResponseStatusSuccess)
					res.SetPayload(make([]byte, 0))
				} else {
					res.SetStatus(tcp.ResponseStatusSuccess)
					var payload = strconv.FormatUint(uint64(qi.Priority), 10) + ":" + qi.Id.String()
					res.SetPayload([]byte(payload))
				}
				break
			case 0x5: // Pop
				qi, err := hp.actualHeap.PopItem()
				if err != nil {
					res.SetStatus(tcp.ResponseStatusFail)
					res.SetPayload(make([]byte, 0))
				} else {
					res.SetStatus(tcp.ResponseStatusSuccess)
					var payload = strconv.FormatUint(uint64(qi.Priority), 10) + ":" + qi.Id.String()
					res.SetPayload([]byte(payload))
				}
				break
			case 0x6: // Push
				var str = string(rcv.request.GetPayload())
				id, err1 := uuid.Parse(strings.Split(str, ":")[1])
				num, err2 := strconv.ParseInt(strings.Split(str, ":")[0], 10, 32)

				if err1 != nil || err2 != nil {
					res.SetStatus(tcp.ResponseStatusFail)
					res.SetPayload(make([]byte, 0))
					break
				}
				qi := &queue.KokaqItem{
					Id:       id,
					Priority: int(num),
				}
				err := hp.actualHeap.PushItem(qi)
				if err != nil {
					res.SetStatus(tcp.ResponseStatusFail)
					res.SetPayload(make([]byte, 0))
				} else {
					res.SetStatus(tcp.ResponseStatusSuccess)
					res.SetPayload([]byte(str))
				}
				break
			default:
				res.SetStatus(tcp.ResponseStatusSuccess)
				res.SetPayload(make([]byte, 0))
				break
			}
			rcv.responseChannel <- res
		}
	}
}
