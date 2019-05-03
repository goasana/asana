package apiauth

import (
	"net/url"
	"testing"
)

func TestSignature(t *testing.T) {
	appSecret := "asana secret"
	method := "GET"
	RequestURL := "http://localhost/test/url"
	params := make(url.Values)
	params.Add("arg1", "hello")
	params.Add("arg2", "asana")

	expectedSignature := "CQ7L9nsOUZ8fSUy7MbnZVxJ7JEPGZRLjGgneFjzZi3g="
	signature := Signature(appSecret, method, params, RequestURL)
	if  signature != expectedSignature {
		t.Error("Signature error")
	}
}
