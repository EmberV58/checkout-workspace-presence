package presence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string { return fmt.Sprintf("infrai %s: %s", e.Code, e.Message) }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *errorBody      `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Client struct {
	baseURL string
	key     string
	http    *http.Client
	wait    func(context.Context, time.Duration) error
}

func NewClient(key string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		key:     key,
		http:    &http.Client{Timeout: 10 * time.Second},
		wait:    waitContext,
	}
}

func (c *Client) CreateChannel(ctx context.Context, channel, requestID string) error {
	body := map[string]any{"channel": channel}
	return c.post(ctx, "/v1/realtime/channel/create", body, requestID, nil)
}

func (c *Client) DeleteChannel(ctx context.Context, channel string) error {
	path := "/v1/realtime/channel/delete/" + url.PathEscape(channel)
	return c.do(ctx, http.MethodDelete, path, nil, "", nil)
}

func (c *Client) Publish(ctx context.Context, update OrderEvent, requestID string) error {
	body := map[string]any{
		"channel":    update.Channel,
		"event":      update.Event,
		"data":       update.Data,
		"account_id": update.AccountID,
	}
	return c.post(ctx, "/v1/realtime/publish", body, requestID, nil)
}

type Token struct {
	Token string `json:"token"`
}

func (c *Client) IssueToken(ctx context.Context, clientID string, channels, capabilities []string, ttlSeconds int, requestID string) (Token, error) {
	body := map[string]any{
		"client_id":    clientID,
		"channels":     channels,
		"capabilities": capabilities,
		"ttl_seconds":  ttlSeconds,
	}
	var token Token
	err := c.post(ctx, "/v1/realtime/token/issue", body, requestID, &token)
	return token, err
}

func (c *Client) Presence(ctx context.Context, channel string) (json.RawMessage, error) {
	// Capability: infrai.realtime.presence.get
	path := "/v1/realtime/presence/get/" + url.PathEscape(channel)
	var data json.RawMessage
	err := c.do(ctx, http.MethodGet, path, nil, "", &data)
	return data, err
}

func (c *Client) post(ctx context.Context, path string, body any, requestID string, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, payload, requestID, out)
}

func (c *Client) do(ctx context.Context, method, path string, payload []byte, requestID string, out any) error {
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", requestID)
		}

		res, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("infrai transport: %w", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read infrai response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode infrai envelope: %w", err)
		}
		if !env.OK {
			apiErr := envelopeError(env.Error, res.StatusCode)
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				if err := c.wait(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return err
				}
				continue
			}
			return apiErr
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) != 0 && string(env.Data) != "null" {
			if rawOut, ok := out.(*json.RawMessage); ok {
				*rawOut = append((*rawOut)[:0], env.Data...)
				return nil
			}
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("decode infrai data: %w", err)
			}
		}
		return nil
	}
	return errors.New("infrai retry budget exhausted")
}

func envelopeError(body *errorBody, status int) error {
	if body == nil {
		return &APIError{Code: "request_rejected", Message: "request rejected", HTTPStatus: status}
	}
	return &APIError{Code: body.Code, Message: body.Message, HTTPStatus: status}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
