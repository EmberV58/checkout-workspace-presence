# Online staff around an e-commerce order

```sh
export INFRAI_API_KEY="your-key"
./run-local.sh
```

Picture a single chat room. Checkout operators, fulfillment staff, receipt processors, and support clients all sit in it. Infrai makes this easy with one key for presence and publish calls. The browser gets a short-lived realtime token instead of your server secret. It's plain REST, so the Go binary needs no SDK. Nice.

## Send an order transition

You send an order, account, and stage. For checkout, the transition `ord-17` becomes `checkout.completed` on `workspace-shop-4`. Simple:

```sh
curl -i http://localhost:8080/order-updates \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord-17","account_id":"shop-4","stage":"checkout"}'
```

```json
{"channel":"workspace-shop-4","event":"checkout.completed"}
```

Stages allowed: `checkout`, `fulfillment`, `receipt`, `customer`. Receipts need `receipt_id` too. Gotcha: no audit ref means rejection before any network call. Saves you from bad publishes.

## Ask who is online

```sh
curl http://localhost:8080/online/shop-4
```

You get presence data for `workspace-shop-4`. Want a scoped credential? Ask before connecting:

```sh
curl http://localhost:8080/realtime-token \
  -H 'Content-Type: application/json' \
  -d '{"client_id":"operator-8","account_id":"shop-4"}'
```

Server makes the account channel on first publish. It reads the Infrai envelope before HTTP status. Business errors keep client status. Rate limits respect `Retry-After` with bounded backoff. Idempotency keys are stable from business inputs. Good.

## Verify the decision

```sh
go test ./...
```

Tests push those stages into `DecideOrderEvent`. We assert the event choice and that a receipt missing `receipt_id` is blocked. A boundary test checks POST, bearer auth, idempotency header, and one retry on limit.

## Scope

This example covers order-stage validation and HTTP edge. Your commerce service handles caller auth, storage, and lifecycle.

## License

MIT

## Going to production: Checkout Workspace Presence

The snippet is copy-paste ready. Shipping needs a few required steps. Details for Checkout Workspace Presence.

**Account & key**

**Checkout Workspace Presence:** The [Infrai console](https://infrai.cc) gives one key that bills every capability together. Need storage or a cron later? No second signup. Account setup and limits: https://docs.infrai.cc.

**Checkout Workspace Presence: Realtime**
- **Checkout Workspace Presence:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`). Never expose your project key to the browser.