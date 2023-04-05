package shortloopfiltertestmode

import (
	. "github.com/short-loop/shortloop-go/common/models/data"
	"net/http"
)

var currentShortloopFilterTestMode *ShortloopFilterTestMode = &ShortloopFilterTestMode{}

type ShortloopFilterTestMode struct {
	ApiProcessor        *ApiProcessor
	UserApplicationName string
}

func CurrentShortloopFilterTestMode() *ShortloopFilterTestMode {
	return currentShortloopFilterTestMode
}

func (sf *ShortloopFilterTestMode) Init() {
	for i := 0; i < 1; i++ {
		go sf.ApiProcessor.bm.PrimarySyncer()
	}
	for i := 0; i < 4; i++ {
		go sf.ApiProcessor.bm.SecondarySyncer()
	}
}

func (sf *ShortloopFilterTestMode) GetObservedApiFromRequest(r *http.Request) ObservedApi {
	return NewObservedApi(r.URL.Path, HTTPRequestMethod(r.Method))
}

func (sf *ShortloopFilterTestMode) GetApiProcessor() *ApiProcessor {
	return sf.ApiProcessor
}

func (sf *ShortloopFilterTestMode) SetApiProcessor(apiProcessor *ApiProcessor) {
	sf.ApiProcessor = apiProcessor
}

func (sf *ShortloopFilterTestMode) GetUserApplicationName() string {
	return sf.UserApplicationName
}

func (sf *ShortloopFilterTestMode) SetUserApplicationName(userApplicationName string) {
	sf.UserApplicationName = userApplicationName
}
