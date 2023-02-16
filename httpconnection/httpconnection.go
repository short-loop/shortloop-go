package httpconnection

import (
	"github.com/short-loop/shortloop-go/sdklogger"
	"github.com/short-loop/shortloop-go/sdkversion"
	"net/http"
)

var AuthKey = ""
var Environment = ""

func SendRequest(httpClient *http.Client, httpRequest *http.Request) (*http.Response, error) {
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set(sdkversion.MAJOR_VERSION_KEY, sdkversion.MAJOR_VERSION)
	httpRequest.Header.Set(sdkversion.MINOR_VERSION_KEY, sdkversion.MINOR_VERSION)
	httpRequest.Header.Set("sdkType", sdkversion.SdkType)
	if AuthKey != "" {
		httpRequest.Header.Set("authKey", AuthKey)
	}
	if Environment != "" {
		httpRequest.Header.Set("environment", Environment)
	}
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		sdklogger.Logger.ErrorF("Error sending request: %s\n", err.Error())
		return nil, err
	}
	return response, nil
}
