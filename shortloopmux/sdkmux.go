package shortloopmux

import (
	"fmt"
	"github.com/short-loop/shortloop-go/buffer"
	. "github.com/short-loop/shortloop-go/common/models/data"
	"github.com/short-loop/shortloop-go/config"
	"github.com/short-loop/shortloop-go/httpconnection"
	"github.com/short-loop/shortloop-go/sdklogger"
	"github.com/short-loop/shortloop-go/sdkversion"
	"github.com/short-loop/shortloop-go/shortloopfilter"
	"github.com/short-loop/shortloop-go/shortloopfiltertestmode"
	"net/http"
	"strings"
)

type Options struct {
	ShortloopEndpoint string
	ApplicationName   string
	LoggingEnabled    bool
	LogLevel          string
	AuthKey           string
	Environment       string
	Capture           string
	MaskHeaders       []string
}

type ShortloopMux interface {
	Filter(h http.Handler) http.Handler
}

type ShortloopMuxNormalMode struct {
	shortloopFilter *shortloopfilter.ShortloopFilter
}

type ShortloopMuxTestMode struct {
	shortloopFilter *shortloopfiltertestmode.ShortloopFilterTestMode
}

func (shortloopMux *ShortloopMuxNormalMode) Filter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if nil == shortloopMux {
			h.ServeHTTP(w, r)
			return
		}

		sf := shortloopMux.shortloopFilter

		if nil == sf {
			h.ServeHTTP(w, r)
			return
		}

		var agentConfigLocal *AgentConfig = sf.AgentConfig

		if nil == agentConfigLocal {
			h.ServeHTTP(w, r)
			return
		}

		if !agentConfigLocal.GetCaptureApiSample() {
			h.ServeHTTP(w, r)
			return
		}

		var observedApi ObservedApi = sf.GetObservedApiFromRequest(r)
		if sf.IsBlackListedApi(observedApi, *agentConfigLocal) {
			h.ServeHTTP(w, r)
			return
		}

		nrw := NewResponseWriterWrapper(w)
		context := shortloopfilter.NewRequestResponseContext(nrw, r, sf.UserApplicationName)
		context.SetObservedApi(observedApi)
		context.SetAgentConfig(*agentConfigLocal)

		var apiConfig *ApiConfig = sf.GetApiConfig(observedApi, *agentConfigLocal)

		if apiConfig != nil {
			context.SetApiConfig(apiConfig)
			context.SetApiBufferKey(buffer.GetApiBufferKeyFromApiConfig(*context.GetApiConfig()))
			sdklogger.Logger.InfoF("ApiConfig found for observedApi: %+v\n", observedApi.GetUri())
			sf.ApiProcessor.ProcessRegisteredApi(context, func(canOffer bool, responsePayloadCaptureAttempted bool) {
				if canOffer {
					nrw.SetShouldCaptureBody(responsePayloadCaptureAttempted)
					h.ServeHTTP(nrw, context.GetHttpRequest())
				} else {
					h.ServeHTTP(w, context.GetHttpRequest())
				}
				return
			}, sf.GetMaskHeaders())
		} else {
			sdklogger.Logger.InfoF("ApiConfig not found for observedApi: %+v\n", observedApi.GetUri())
			context.SetApiBufferKey(buffer.GetApiBufferKeyFromObservedApi(context.GetObservedApi()))
			sf.ApiProcessor.ProcessDiscoveredApi(context, func(canOffer bool) {
				if canOffer {
					h.ServeHTTP(nrw, context.GetHttpRequest())
				} else {
					h.ServeHTTP(w, context.GetHttpRequest())
				}
				return
			}, sf.GetMaskHeaders())
		}

		// h.ServeHTTP(nrw, r)
	})
}

func (shortloopMux *ShortloopMuxTestMode) Filter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if nil == shortloopMux {
			h.ServeHTTP(w, r)
			return
		}

		sf := shortloopMux.shortloopFilter

		if nil == sf {
			h.ServeHTTP(w, r)
			return
		}

		var observedApi ObservedApi = sf.GetObservedApiFromRequest(r)

		nrw := NewResponseWriterWrapper(w)
		context := shortloopfilter.NewRequestResponseContext(nrw, r, sf.UserApplicationName)
		context.SetObservedApi(observedApi)
		sdklogger.Logger.InfoF("Processing Api: %+v\n", observedApi.GetUri())
		sf.ApiProcessor.ProcessApi(context, func(canOffer bool, responsePayloadCaptureAttempted bool) {
			if canOffer {
				nrw.SetShouldCaptureBody(responsePayloadCaptureAttempted)
				h.ServeHTTP(nrw, context.GetHttpRequest())
			} else {
				h.ServeHTTP(w, context.GetHttpRequest())
			}
			return
		}, sf.GetMaskHeaders())
	})
}

func Init(options Options) (ShortloopMux, error) {
	options.ShortloopEndpoint = strings.TrimSpace(options.ShortloopEndpoint)
	options.ApplicationName = strings.TrimSpace(options.ApplicationName)
	options.AuthKey = strings.TrimSpace(options.AuthKey)
	options.Environment = strings.TrimSpace(options.Environment)
	options.Capture = strings.TrimSpace(options.Capture)

	if options.AuthKey == "" {
		return nil, fmt.Errorf("AuthKey is required")
	}
	if options.Environment == "" {
		return nil, fmt.Errorf("Environment is required")
	}
	if options.ShortloopEndpoint == "" {
		return nil, fmt.Errorf("ShortloopEndpoint is required")
	}
	if options.ApplicationName == "" {
		return nil, fmt.Errorf("ApplicationName is required")
	}
	httpconnection.AuthKey = options.AuthKey
	httpconnection.Environment = options.Environment

	loggingEnabled := options.LoggingEnabled
	logLevel := "ERROR"
	if options.LogLevel != "" {
		logLevel = options.LogLevel
	}

	if options.MaskHeaders == nil {
		options.MaskHeaders = []string{}
	}

	sdklogger.Logger.SetLoggingEnabled(loggingEnabled)
	sdklogger.Logger.SetLogLevel(sdklogger.GetLogLevel(logLevel))

	sdklogger.Logger.Info("Initializing Shortloop SDK")
	if strings.EqualFold(options.Capture, "always") {
		sdklogger.Logger.Info("Shortloop SDK is running in test mode to sample 100% requests")
		shortloopMux := &ShortloopMuxTestMode{shortloopfiltertestmode.CurrentShortloopFilterTestMode()}

		bufferManager := shortloopfiltertestmode.NewBufferManager(options.ShortloopEndpoint, http.Client{})
		apiProcessor := shortloopfiltertestmode.NewApiProcessor(&bufferManager)

		shortloopFilter := shortloopfiltertestmode.CurrentShortloopFilterTestMode()
		shortloopFilter.SetUserApplicationName(options.ApplicationName)
		shortloopFilter.SetApiProcessor(apiProcessor)
		shortloopFilter.SetMaskHeaders(options.MaskHeaders)
		shortloopFilter.Init()
		return shortloopMux, nil
	}
	shortloopMux := &ShortloopMuxNormalMode{shortloopfilter.CurrentShortloopFilter()}
	configManager := config.CurrentConfigManager()
	configManager.SetCtUrl(options.ShortloopEndpoint)
	configManager.SetUserApplicationName(options.ApplicationName)
	configManager.SetHttpClient(http.Client{})
	configManager.Init()

	discoveredApiManager := buffer.NewDiscoveredApiManager(configManager, http.Client{}, options.ShortloopEndpoint)
	discoveredApiManager.Init()
	registeredApiManager := buffer.NewRegisteredApiManager(configManager, http.Client{}, options.ShortloopEndpoint)
	registeredApiManager.Init()
	apiProcessor := shortloopfilter.NewApiProcessor(discoveredApiManager, registeredApiManager)

	shortloopFilter := shortloopfilter.CurrentShortloopFilter()
	shortloopFilter.SetUserApplicationName(options.ApplicationName)
	shortloopFilter.SetConfigManager(configManager)
	shortloopFilter.SetApiProcessor(apiProcessor)
	shortloopFilter.SetMaskHeaders(options.MaskHeaders)
	shortloopFilter.Init()
	sdkversion.SdkType = "Go-Mux"
	sdklogger.Logger.Info("Initialized Shortloop SDK")
	sdklogger.Logger.InfoF("Initialized Shortloop SDK\napplication name: %v\nurl: %v\nagent id:%v\nSDK Version: %v.%v\nSDKType: %v\n",
		options.ApplicationName,
		options.ShortloopEndpoint,
		configManager.GetAgentId(),
		sdkversion.MAJOR_VERSION,
		sdkversion.MINOR_VERSION,
		sdkversion.SdkType)
	return shortloopMux, nil
}
