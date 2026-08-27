package presence

import "fmt"

type OrderUpdate struct {
	OrderID   string `json:"order_id"`
	AccountID string `json:"account_id"`
	Stage     string `json:"stage"`
	ReceiptID string `json:"receipt_id,omitempty"`
}

type OrderEvent struct {
	Channel   string
	Event     string
	Data      map[string]string
	AccountID string
}

func DecideOrderEvent(update OrderUpdate) (OrderEvent, error) {
	eventByStage := map[string]string{
		"checkout":    "checkout.completed",
		"fulfillment": "fulfillment.started",
		"receipt":     "receipt.issued",
		"customer":    "customer.order_updated",
	}
	event, ok := eventByStage[update.Stage]
	if !ok {
		return OrderEvent{}, fmt.Errorf("unsupported order stage %q", update.Stage)
	}
	if update.OrderID == "" || update.AccountID == "" {
		return OrderEvent{}, fmt.Errorf("order_id and account_id are required")
	}
	if update.Stage == "receipt" && update.ReceiptID == "" {
		return OrderEvent{}, fmt.Errorf("receipt_id is required for receipt stage")
	}
	data := map[string]string{"order_id": update.OrderID, "stage": update.Stage}
	if update.ReceiptID != "" {
		data["receipt_id"] = update.ReceiptID
	}
	return OrderEvent{
		Channel:   "workspace-" + update.AccountID,
		Event:     event,
		Data:      data,
		AccountID: update.AccountID,
	}, nil
}
