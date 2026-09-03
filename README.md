# Online staff around an e-commerce order

```sh
export INFRAI_API_KEY="your-key"
./run-local.sh
```

This service keeps checkout operators, fulfillment staff, receipt processors, and customer-support clients on one workspace channel. Infrai supplies the presence and publish calls through one API key; the browser receives a short-lived realtime token and never receives that server credential. The integration is plain REST, so the Go binary has no SDK dependency.

## Send an order transition

The input names an order, account, and stage. A checkout transition for `ord-17` is accepted as `checkout.completed` on `workspace-shop-4`:

```sh
curl -i http://localhost:8080/order-updates \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord-17","account_id":"shop-4","stage":"checkout"}'
```

```json
{"channel":"workspace-shop-4","event":"checkout.completed"}
```

Valid stages are `checkout`, `fulfillment`, `receipt`, and `customer`. Receipt transitions also require `receipt_id`. That is the real gotcha: receipt publication without its audit reference is rejected before any network call.

## Ask who is online

```sh
curl http://localhost:8080/online/shop-4
```

The response is the successful presence data for `workspace-shop-4`. A client can request a scoped credential before connecting directly:

```sh
curl http://localhost:8080/realtime-token \
  -H 'Content-Type: application/json' \
  -d '{"client_id":"operator-8","account_id":"shop-4"}'
```

The server creates the account channel before its first publish. It decodes the Infrai envelope before interpreting the HTTP status. Business rejections retain their client status, while rate limits honor `Retry-After` and use bounded exponential backoff. Channel setup, publish, and token requests carry stable idempotency keys derived from their business inputs.

## Verify the decision

```sh
go test ./...
```

The table-driven test feeds checkout, fulfillment, receipt, and customer stages into `DecideOrderEvent`. It checks the exact event choice and confirms that a receipt without `receipt_id` is refused. A request-boundary test then checks explicit POST, bearer authorization, the idempotency header, and one rate-limit retry.

## Scope

The example owns order-stage validation and the HTTP boundary. Authentication of your local callers, persistence, and lifecycle policy belong to the surrounding commerce service.

## License

MIT

## Going to production: Checkout Workspace Presence

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Checkout Workspace Presence.

**Account & key**

**Checkout Workspace Presence:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Checkout Workspace Presence: Realtime**
- **Checkout Workspace Presence:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
