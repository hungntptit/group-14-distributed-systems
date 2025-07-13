package network

import "net/http"

func Forward(method, url string, body []byte) (*http.Response, error) {
	return nil, nil
}
func CopyResponse(w http.ResponseWriter, resp *http.Response) {
}
