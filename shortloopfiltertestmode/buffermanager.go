package shortloopfiltertestmode

import (
	"bytes"
	"encoding/json"
	"github.com/short-loop/shortloop-go/common/models/data"
	"github.com/short-loop/shortloop-go/httpconnection"
	"github.com/short-loop/shortloop-go/sdklogger"
	"net/http"
	"time"
)

type BufferManager struct {
	sb         SimpleBuffer
	httpClient http.Client
	ctUrl      string
}

func NewBufferManager(ctUrl string, httpClient http.Client) BufferManager {
	return BufferManager{
		sb:         NewSimpleBuffer(5, 20),
		httpClient: httpClient,
		ctUrl:      ctUrl,
	}
}

func (bm *BufferManager) SecondarySyncer() {
	defer func() {
		if err := recover(); err != nil {
			sdklogger.Logger.ErrorF("Panic in BufferManager::SecondarySyncer: %s\n", err)
		}
	}()
	for {
		apiSamples := bm.sb.WaitForSamples()
		bm.Sync(apiSamples[:])
	}
}

func (bm *BufferManager) PrimarySyncer() {
	defer func() {
		if err := recover(); err != nil {
			sdklogger.Logger.ErrorF("Panic in BufferManager::PrimarySyncer: %s\n", err)
		}
	}()
	for {
		apiSample := bm.sb.WaitForSample()
		bm.Sync([]*data.APISample{apiSample})
	}
}

func (bm *BufferManager) Offer(e *data.APISample) bool {
	return bm.sb.Offer(e)
}

func (bm *BufferManager) GetUri() string {
	return "/api/v2/data-ingestion/api-sample"
}

func (bm *BufferManager) Sync(apiSampleList []*data.APISample) {
	defer func() {
		if err := recover(); err != nil {
			sdklogger.Logger.ErrorF("Panic in TestMode BufferManager::Sync: %s\n", err)
		}
	}()

	if nil == apiSampleList {
		return
	}

	sdklogger.Logger.InfoF("Syncing api samples")
	jsonBodyContent, err := json.Marshal(apiSampleList)
	if err != nil {
		sdklogger.Logger.ErrorF("Error marshalling api sample list: %s\n", err.Error())
	}

	bodyReader := bytes.NewReader(jsonBodyContent)
	req, err := http.NewRequest("POST", bm.ctUrl+bm.GetUri(), bodyReader)
	if err != nil {
		sdklogger.Logger.ErrorF("Error creating request: %s\n", err.Error())
	}

	var startTime = time.Now()
	response, err := httpconnection.SendRequest(&bm.httpClient, req)
	if err != nil {
		sdklogger.Logger.ErrorF("Error sending api samples: %s\n", err.Error())
	}
	err = response.Body.Close()
	sdklogger.Logger.InfoF("request duration - %v", time.Since(startTime).Milliseconds())

	if err != nil {
		sdklogger.Logger.ErrorF("Error closing connection: %s\n", err.Error())
	}
	sdklogger.Logger.InfoF("Finished Syncing api samples")
}
