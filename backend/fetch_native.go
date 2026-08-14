//go:build !wasip1

package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type fetchHostRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type fetchHostResponse struct {
	Status  int               `json:"status"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Error   string            `json:"error"`
}

var mockHTTPClient func(req fetchHostRequest) (*fetchHostResponse, error)

func doFetch(req fetchHostRequest) (*fetchHostResponse, error) {
	if mockHTTPClient != nil {
		return mockHTTPClient(req)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	httpReq, err := http.NewRequest(method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	hdrs := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			hdrs[k] = v[0]
		}
	}

	res := &fetchHostResponse{
		Status:  resp.StatusCode,
		Body:    string(respBytes),
		Headers: hdrs,
	}

	if resp.StatusCode >= 400 {
		return res, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	return res, nil
}
