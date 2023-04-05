package shortloopfiltertestmode

import (
	"github.com/short-loop/shortloop-go/common/models/data"
)

type SimpleBuffer struct {
	primaryChannel   chan *data.APISample
	secondaryChannel chan *data.APISample
}

func NewSimpleBuffer(size1 int, size2 int) SimpleBuffer {
	return SimpleBuffer{
		primaryChannel:   make(chan *data.APISample, size1),
		secondaryChannel: make(chan *data.APISample, size2),
	}
}

func (sb *SimpleBuffer) Offer(e *data.APISample) bool {
	select {
	case sb.primaryChannel <- e:
		return true
	case sb.secondaryChannel <- e:
		return true
	default:
		return false
	}
}

func (sb *SimpleBuffer) WaitForSamples() [5]*data.APISample {
	var samples [5]*data.APISample
	for i := 0; i < 5; i++ {
		samples[i] = <-sb.secondaryChannel
	}
	return samples
}

func (sb *SimpleBuffer) WaitForSample() *data.APISample {
	return <-sb.primaryChannel
}

func (sb *SimpleBuffer) GetContentCount() int {
	return len(sb.primaryChannel)
}
