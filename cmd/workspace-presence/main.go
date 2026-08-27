package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	presence "example.com/checkout-presence"
)

type server struct {
	client   *presence.Client
	channels sync.Map
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	s := &server{client: presence.NewClient(key)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /order-updates", s.orderUpdate)
	mux.HandleFunc("GET /online/{accountID}", s.online)
	mux.HandleFunc("POST /realtime-token", s.realtimeToken)
	httpServer := &http.Server{Addr: ":8080", Handler: mux}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Println("workspace presence listening on :8080")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server: %v", err)
			stop()
		}
	}()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	s.deleteChannels(shutdownCtx)
}

func (s *server) orderUpdate(w http.ResponseWriter, r *http.Request) {
	var input presence.OrderUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	event, err := presence.DecideOrderEvent(input)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	requestID := input.OrderID + "-" + input.Stage
	if err := s.client.CreateChannel(r.Context(), event.Channel, "channel-"+input.AccountID); err != nil {
		writeUpstreamError(w, err)
		return
	}
	s.channels.Store(event.Channel, struct{}{})
	if err := s.client.Publish(r.Context(), event, requestID); err != nil {
		writeUpstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"channel": event.Channel, "event": event.Event})
}

func (s *server) deleteChannels(ctx context.Context) {
	s.channels.Range(func(key, _ any) bool {
		channel := key.(string)
		if err := s.client.DeleteChannel(ctx, channel); err != nil {
			log.Printf("delete channel %s: %v", channel, err)
		} else {
			s.channels.Delete(channel)
		}
		return true
	})
}

func (s *server) online(w http.ResponseWriter, r *http.Request) {
	data, err := s.client.Presence(r.Context(), "workspace-"+r.PathValue("accountID"))
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *server) realtimeToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ClientID  string `json:"client_id"`
		AccountID string `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ClientID == "" || input.AccountID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id and account_id are required"})
		return
	}
	requestID := input.ClientID + "-" + input.AccountID
	token, err := s.client.IssueToken(r.Context(), input.ClientID, []string{"workspace-" + input.AccountID}, []string{"subscribe", "presence"}, 900, requestID)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, token)
}

func writeUpstreamError(w http.ResponseWriter, err error) {
	var apiErr *presence.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		writeJSON(w, apiErr.HTTPStatus, map[string]string{"error": apiErr.Code, "message": apiErr.Message})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
