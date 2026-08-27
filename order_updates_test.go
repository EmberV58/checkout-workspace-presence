package presence

import "testing"

func TestDecideOrderEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     OrderUpdate
		wantEvent string
		wantError bool
	}{
		{"checkout", OrderUpdate{OrderID: "ord-17", AccountID: "shop-4", Stage: "checkout"}, "checkout.completed", false},
		{"fulfillment", OrderUpdate{OrderID: "ord-17", AccountID: "shop-4", Stage: "fulfillment"}, "fulfillment.started", false},
		{"receipt", OrderUpdate{OrderID: "ord-17", AccountID: "shop-4", Stage: "receipt", ReceiptID: "rcpt-9"}, "receipt.issued", false},
		{"customer update", OrderUpdate{OrderID: "ord-17", AccountID: "shop-4", Stage: "customer"}, "customer.order_updated", false},
		{"receipt requires reference", OrderUpdate{OrderID: "ord-17", AccountID: "shop-4", Stage: "receipt"}, "", true},
		{"unknown stage", OrderUpdate{OrderID: "ord-17", AccountID: "shop-4", Stage: "refund"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecideOrderEvent(tt.input)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError %v", err, tt.wantError)
			}
			if got.Event != tt.wantEvent {
				t.Fatalf("event = %q, want %q", got.Event, tt.wantEvent)
			}
			if !tt.wantError && got.Channel != "workspace-shop-4" {
				t.Fatalf("channel = %q", got.Channel)
			}
		})
	}
}
