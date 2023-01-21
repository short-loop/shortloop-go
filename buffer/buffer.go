package buffer

import (
	"github.com/short-loop/shortloop-go/common/models/data"
)

type Buffer interface {
	Offer(e *data.APISample) bool
	CanOffer() bool
	Poll() *data.APISample
	Clear() bool
	GetContentCount() int
}
