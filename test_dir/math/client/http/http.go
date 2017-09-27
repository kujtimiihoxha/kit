package http

import (
	bytes "bytes"
	context "context"
	json "encoding/json"
	"errors"
	ioutil "io/ioutil"
	http1 "net/http"
	url "net/url"
	strings "strings"

	endpoint "github.com/go-kit/kit/endpoint"
	http "github.com/go-kit/kit/transport/http"
	endpoint1 "github.com/kujtimiihoxha/kit/test_dir/math/pkg/endpoint"
	http2 "github.com/kujtimiihoxha/kit/test_dir/math/pkg/http"
	service "github.com/kujtimiihoxha/kit/test_dir/math/pkg/service"
)

// New returns an AddService backed by an HTTP server living at the remote
// instance. We expect instance to come from a service discovery system, so
// likely of the form "host:port".
func New(instance string, options map[string][]http.ClientOption) (service.MathService, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}
	var sumEndpoint endpoint.Endpoint
	{
		sumEndpoint = http.NewClient("POST", copyURL(u, "/sum"), encodeHTTPGenericRequest, decodeSumResponse, options["Sum"]...).Endpoint()
	}

	var prodEndpoint endpoint.Endpoint
	{
		prodEndpoint = http.NewClient("POST", copyURL(u, "/prod"), encodeHTTPGenericRequest, decodeProdResponse, options["Prod"]...).Endpoint()
	}

	return endpoint1.Endpoints{
		ProdEndpoint: prodEndpoint,
		SumEndpoint:  sumEndpoint,
	}, nil
}

// EncodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that
// SON-encodes any request to the request body. Primarily useful in a client.
func encodeHTTPGenericRequest(_ context.Context, r *http1.Request, request interface{}) error {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// decodeSumResponse is a transport/http.DecodeResponseFunc that decodes
// a JSON-encoded concat response from the HTTP response body. If the response
// as a non-200 status code, we will interpret that as an error and attempt to
//  decode the specific error message from the response body.
func decodeSumResponse(_ context.Context, r *http1.Response) (interface{}, error) {
	if r.StatusCode != http1.StatusOK {
		return nil, http2.ErrorDecoder(r)
	}
	var resp endpoint1.SumResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// decodeProdResponse is a transport/http.DecodeResponseFunc that decodes
// a JSON-encoded concat response from the HTTP response body. If the response
// as a non-200 status code, we will interpret that as an error and attempt to
//  decode the specific error message from the response body.
func decodeProdResponse(_ context.Context, r *http1.Response) (interface{}, error) {
	if r.StatusCode != http1.StatusOK {
		if r.StatusCode == http1.StatusNotFound {
			return nil, errors.New("resource not found")
		}
		return 0, http2.ErrorDecoder(r)
	}
	var resp endpoint1.ProdResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}
func copyURL(base *url.URL, path string) (next *url.URL) {
	n := *base
	n.Path = path
	next = &n
	return
}
