//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

//go:wasmimport paca fetch
//go:noescape
func hostFetch(reqPtr, reqLen, resPtrPtr, resLenPtr uintptr)

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

func doFetch(req fetchHostRequest) (*fetchHostResponse, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal fetch request: %w", err)
	}

	var resPtr, resLen uint32
	var reqPtr uintptr
	if len(reqJSON) > 0 {
		reqPtr = uintptr(unsafe.Pointer(&reqJSON[0]))
	}

	hostFetch(
		reqPtr,
		uintptr(len(reqJSON)),
		uintptr(unsafe.Pointer(&resPtr)),
		uintptr(unsafe.Pointer(&resLen)),
	)

	if resPtr == 0 || resLen == 0 {
		return nil, fmt.Errorf("fetch returned empty response")
	}

	resBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(resPtr))), resLen)
	var resp fetchHostResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal fetch response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	if resp.Status >= 400 || resp.Status == 0 {
		return &resp, fmt.Errorf("HTTP %d: %s", resp.Status, resp.Body)
	}

	return &resp, nil
}
