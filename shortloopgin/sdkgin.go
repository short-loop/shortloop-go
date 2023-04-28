package shortloopgin

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/short-loop/shortloop-go/buffer"
	. "github.com/short-loop/shortloop-go/common/models/data"
	"github.com/short-loop/shortloop-go/config"
	"github.com/short-loop/shortloop-go/httpconnection"
	"github.com/short-loop/shortloop-go/sdklogger"
	"github.com/short-loop/shortloop-go/sdkversion"
	"github.com/short-loop/shortloop-go/shortloopfilter"
	"github.com/short-loop/shortloop-go/shortloopfiltertestmode"
	"net/http"
	"os"
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

type ShortloopGin interface {
	Filter() gin.HandlerFunc
}

type ShortloopGinNormalMode struct {
	shortloopFilter *shortloopfilter.ShortloopFilter
}

type ShortloopGinTestMode struct {
	shortloopFilter *shortloopfiltertestmode.ShortloopFilterTestMode
}

func (shortloopGin *ShortloopGinNormalMode) Filter() gin.HandlerFunc {
	return func(c *gin.Context) {

		if nil == shortloopGin {
			c.Next()
			return
		}

		sf := shortloopGin.shortloopFilter

		if nil == sf {
			c.Next()
			return
		}

		var agentConfigLocal *AgentConfig = sf.AgentConfig

		if nil == agentConfigLocal {
			c.Next()
			return
		}

		if !agentConfigLocal.GetCaptureApiSample() {
			c.Next()
			return
		}

		var observedApi ObservedApi = sf.GetObservedApiFromRequest(c.Request)
		if sf.IsBlackListedApi(observedApi, *agentConfigLocal) {
			c.Next()
			return
		}
		nrw := NewResponseWriterWrapper(c.Writer)
		//c.Writer = nrw
		// nrw := NewResponseWriterWrapper(w)
		context := shortloopfilter.NewRequestResponseContext(nrw, c.Request, sf.UserApplicationName)
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
					c.Writer = nrw
				}
				c.Next()
				return
			}, sf.GetMaskHeaders())
		} else {
			sdklogger.Logger.InfoF("ApiConfig not found for observedApi: %+v\n", observedApi.GetUri())
			context.SetApiBufferKey(buffer.GetApiBufferKeyFromObservedApi(context.GetObservedApi()))
			sf.ApiProcessor.ProcessDiscoveredApi(context, func(canOffer bool) {
				if canOffer {
					c.Writer = nrw
				}
				c.Next()
				return
			}, sf.GetMaskHeaders())
		}
	}
}

func (shortloopGin *ShortloopGinTestMode) Filter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if nil == shortloopGin {
			c.Next()
			return
		}

		sf := shortloopGin.shortloopFilter

		if nil == sf {
			c.Next()
			return
		}

		var observedApi ObservedApi = sf.GetObservedApiFromRequest(c.Request)

		nrw := NewResponseWriterWrapper(c.Writer)
		context := shortloopfilter.NewRequestResponseContext(nrw, c.Request, sf.UserApplicationName)
		context.SetObservedApi(observedApi)
		sdklogger.Logger.InfoF("Processing Api: %+v\n", observedApi.GetUri())
		sf.ApiProcessor.ProcessApi(context, func(canOffer bool, responsePayloadCaptureAttempted bool) {
			if canOffer {
				nrw.SetShouldCaptureBody(responsePayloadCaptureAttempted)
				c.Writer = nrw
			}
			c.Next()
			return
		}, sf.GetMaskHeaders())
	}
}

func Init(options Options) (ShortloopGin, error) {

	if os.Getenv("GOARCH") == "386" {
		return nil, fmt.Errorf("32 bit Arch not supported by shortloop sdk")
	}

	shortloopGin := &ShortloopGinNormalMode{shortloopfilter.CurrentShortloopFilter()}

	options.ShortloopEndpoint = strings.TrimSpace(options.ShortloopEndpoint)
	options.ApplicationName = strings.TrimSpace(options.ApplicationName)
	options.AuthKey = strings.TrimSpace(options.AuthKey)
	options.Environment = strings.TrimSpace(options.Environment)

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
		shortloopGin := &ShortloopGinTestMode{shortloopfiltertestmode.CurrentShortloopFilterTestMode()}

		bufferManager := shortloopfiltertestmode.NewBufferManager(options.ShortloopEndpoint, http.Client{})
		apiProcessor := shortloopfiltertestmode.NewApiProcessor(&bufferManager)

		shortloopFilter := shortloopfiltertestmode.CurrentShortloopFilterTestMode()
		shortloopFilter.SetUserApplicationName(options.ApplicationName)
		shortloopFilter.SetApiProcessor(apiProcessor)
		shortloopFilter.SetMaskHeaders(options.MaskHeaders)
		shortloopFilter.Init()
		return shortloopGin, nil
	}

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
	sdkversion.SdkType = "Go-Gin"
	sdklogger.Logger.InfoF("Initialized Shortloop SDK\napplication name: %v\nurl: %v\nagent id:%v\nSDK Version: %v.%v\nSDKType: %v\n",
		options.ApplicationName,
		options.ShortloopEndpoint,
		configManager.GetAgentId(),
		sdkversion.MAJOR_VERSION,
		sdkversion.MINOR_VERSION,
		sdkversion.SdkType)
	return shortloopGin, nil
}
