package shortloopfiltertestmode

import (
	"bytes"
	. "github.com/short-loop/shortloop-go/common/models/data"
	"github.com/short-loop/shortloop-go/sdklogger"
	"github.com/short-loop/shortloop-go/shortloopfilter"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ApiProcessor struct {
	bm        *BufferManager
	dropCount int
}

func NewApiProcessor(bm *BufferManager) *ApiProcessor {
	return &ApiProcessor{
		bm: bm,
	}
}

func (ap *ApiProcessor) ProcessApi(context shortloopfilter.RequestResponseContext, next func(canOffer bool, responsePayloadCaptureAttempted bool)) {
	if ap.bm == nil {
		sdklogger.Logger.Error("BufferManager is nil inside ProcessApi")
		next(false, false)
		return
	}

	context.SetPayloadCaptureAttempted(true)
	context.SetRequestPayload(ap.wrapRequest(context.GetHttpRequest()))
	context.SetRequestPayloadCaptureAttempted(true)
	context.SetResponsePayloadCaptureAttempted(true)

	var startTime time.Time
	startTime = time.Now()

	next(true, true)

	context.SetLatency(time.Since(startTime).Milliseconds())

	ap.tryOffering(context)
}

func (ap *ApiProcessor) tryOffering(context shortloopfilter.RequestResponseContext) {
	defer func() {
		if err := recover(); err != nil {
			sdklogger.Logger.ErrorF("Panic in Test Mode tryOffering: %s\n", err)
		}
	}()

	var apiSample *APISample = ap.getBufferEntryForApiSample(context)
	sdklogger.Logger.InfoF("trying to offer apiSample of %s to buffer in Test Mode\n", apiSample.GetRawUri())
	if apiSample != nil {
		if !ap.bm.Offer(apiSample) {
			if sdklogger.Logger.GetLoggingEnabled() && sdklogger.Logger.GetLogLevel() == sdklogger.INFO {
				ap.dropCount++
				sdklogger.Logger.InfoF("drop count %v", ap.dropCount)
			}
		}
	}
}

func (ap *ApiProcessor) getBufferEntryForApiSample(context shortloopfilter.RequestResponseContext) *APISample {
	var apiSample APISample = APISample{}

	apiSample.SetApplicationName(context.GetApplicationName())
	if nil != context.GetApiConfig() {
		apiSample.SetMethod(context.GetApiConfig().GetMethod())
	} else {
		apiSample.SetMethod(context.GetObservedApi().GetMethod())
	}
	apiSample.SetRawUri(context.GetObservedApi().GetUri().GetURIPath())
	apiSample.SetParameters(ap.getParameters(context.GetHttpRequest()))
	apiSample.SetRequestHeaders(ap.getRequestHeaders(context.GetHttpRequest()))
	apiSample.SetResponseHeaders(ap.getResponseHeaders(context.GetResponseWriterWrapper()))
	apiSample.SetLatency(context.GetLatency())
	var scheme string = ap.getScheme(context.GetHttpRequest())
	apiSample.SetScheme(scheme)
	hostname, port := ap.getHostAndPort(context.GetHttpRequest(), scheme)
	apiSample.SetHostName(hostname)
	apiSample.SetPort(port)

	if context.GetRequestPayloadCaptureAttempted() {
		apiSample.SetRequestPayload(string(context.GetRequestPayload()))
	}

	apiSample.SetStatusCode(context.GetResponseWriterWrapper().Status())
	if context.GetResponsePayloadCaptureAttempted() {
		apiSample.SetResponsePayload(context.GetResponseWriterWrapper().BodyString())
	}

	apiSample.SetPayloadCaptureAttempted(context.GetPayloadCaptureAttempted())
	apiSample.SetRequestPayloadCaptureAttempted(context.GetRequestPayloadCaptureAttempted())
	apiSample.SetResponsePayloadCaptureAttempted(context.GetResponsePayloadCaptureAttempted())

	return &apiSample
}

func (ap *ApiProcessor) getScheme(request *http.Request) string {
	if request.URL.Scheme == "" {
		if request.TLS == nil {
			return "http"
		} else {
			return "https"
		}
	}
	return request.URL.Scheme
}

func (ap *ApiProcessor) getHostAndPort(request *http.Request, scheme string) (hostname string, port int) {
	parts := strings.Split(request.Host, ":")
	if len(parts) > 0 {
		hostname = parts[0]
	}
	if len(parts) > 1 {
		port, _ = strconv.Atoi(parts[1])
	} else {
		if scheme == "http" {
			port = 80
		} else if scheme == "https" {
			port = 443
		}
	}
	return
}

func (ap *ApiProcessor) getRequestHeaders(request *http.Request) map[string]string {
	if request == nil {
		return map[string]string{}
	}

	var headers map[string]string = map[string]string{}

	for key := range request.Header {
		headers[key] = request.Header.Get(key)
	}
	return headers
}

func (ap *ApiProcessor) getResponseHeaders(responseWriterWrapper ResponseWriter) map[string]string {
	if responseWriterWrapper == nil {
		return map[string]string{}
	}

	var headers map[string]string = map[string]string{}

	for key := range responseWriterWrapper.GetHeader() {
		headers[key] = responseWriterWrapper.GetHeader().Get(key)
	}
	return headers
}

func (ap *ApiProcessor) getParameters(request *http.Request) map[string][]string {
	return request.URL.Query()
}

func (ap *ApiProcessor) wrapRequest(request *http.Request) *bytes.Buffer {
	buf := &bytes.Buffer{}
	// Don't buffer if there is no body.
	if request.Body == nil || request.Body == http.NoBody {
		return buf
	}
	request.Body = readCloser{
		Reader: io.TeeReader(request.Body, buf),
		Closer: request.Body,
	}
	return buf
}

// readCloser combines an io.Reader and an io.Closer to implement io.ReadCloser.
type readCloser struct {
	io.Reader
	io.Closer
}
