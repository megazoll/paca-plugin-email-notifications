//go:build !wasip1

package main

import (
	"fmt"
)

// TestSMTPHook allows unit tests to mock or intercept SMTP dispatches.
var TestSMTPHook func(req SMTPHostPayload) error

func sendViaHostSMTP(req SMTPHostPayload) error {
	if TestSMTPHook != nil {
		return TestSMTPHook(req)
	}
	if req.Host == "" {
		return fmt.Errorf("smtp: host is required")
	}
	return nil
}
