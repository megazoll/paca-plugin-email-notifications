//go:build wasip1

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"unsafe"
)

//go:wasmimport paca smtp_send
func hostSMTPSend(reqPtr, reqLen, resPtrPtr, resLenPtr int64)

var smtpOutBuf [8]byte

func sendViaHostSMTP(req SMTPHostPayload) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal smtp request: %w", err)
	}

	ptr := int64(uintptr(unsafe.Pointer(&data[0])))
	length := int64(len(data))
	outPtrPtr := int64(uintptr(unsafe.Pointer(&smtpOutBuf[0])))
	outLenPtr := int64(uintptr(unsafe.Pointer(&smtpOutBuf[4])))

	hostSMTPSend(ptr, length, outPtrPtr, outLenPtr)

	resPtr := binary.LittleEndian.Uint32(smtpOutBuf[0:4])
	resLen := binary.LittleEndian.Uint32(smtpOutBuf[4:8])

	if resLen == 0 {
		return nil
	}

	resBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(resPtr))), resLen)
	var res SMTPHostResult
	if err := json.Unmarshal(resBytes, &res); err != nil {
		return fmt.Errorf("decode smtp response: %w", err)
	}
	if !res.Success {
		if res.Error != "" {
			return fmt.Errorf("%s", res.Error)
		}
		return fmt.Errorf("smtp dispatch failed")
	}
	return nil
}
