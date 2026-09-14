package main

import "testing"

func TestPairingRequestOnlyAcceptsOneOpaquePairRequest(t *testing.T) {
	good, err := pairingRequest("sparstrowgen://pair?request=AbCd-123_")
	if err != nil || good != "AbCd-123_" {
		t.Fatalf("good pairing link: request=%q err=%v", good, err)
	}
	for _, raw := range []string{
		"https://pair?request=AbCd", "sparstrowgen://other?request=AbCd",
		"sparstrowgen://pair", "sparstrowgen://pair?request=AbCd&extra=1",
		"sparstrowgen://pair?request=has%20space",
	} {
		if _, err := pairingRequest(raw); err == nil {
			t.Errorf("accepted unsafe link %q", raw)
		}
	}
}
