package presence

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestPublishRequestBoundary(t *testing.T) {
	var attempts int
	client := NewClient("test-key")
	client.http = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s", req.Method)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer authorization")
		}
		if req.Header.Get("Idempotency-Key") != "order-17-checkout" {
			t.Fatal("missing idempotency key")
		}
		status, body := http.StatusOK, `{"ok":true,"data":{},"error":null,"metadata":{}}`
		if attempts == 1 {
			status, body = http.StatusTooManyRequests, `{"ok":false,"data":null,"error":{"code":"rate_limited","message":"retry later"},"metadata":{}}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	client.wait = func(context.Context, time.Duration) error { return nil }

	event, err := DecideOrderEvent(OrderUpdate{OrderID: "ord-17", AccountID: "shop-4", Stage: "checkout"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Publish(context.Background(), event, "order-17-checkout"); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestDeleteChannelRequestBoundary(t *testing.T) {
	client := NewClient("test-key")
	client.http = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("method = %s", req.Method)
		}
		if req.URL.EscapedPath() != "/v1/realtime/channel/delete/workspace-shop%2F4" {
			t.Fatalf("path = %s", req.URL.EscapedPath())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"data":{},"error":null,"metadata":{}}`)),
		}, nil
	})}

	if err := client.DeleteChannel(context.Background(), "workspace-shop/4"); err != nil {
		t.Fatal(err)
	}
}
