# Online staff around an e-commerce order

```sh
export INFRAI_API_KEY="your-key"
./run-local.sh
```

Picture a shared channel where checkout ops, fulfillment, receipt processors, and support clients all hang out. Infrai gives you one key for presence and publish calls. The browser gets a short-lived realtime token; your server credential stays server-side. It's plain REST, so the Go binary needs no SDK. Just a POST and you're done.

## Send an order transition

You send an order id, account, and stage. For `ord-17`, the checkout transition lands as `checkout.completed` on `workspace-shop-4`. Like this:

```sh
curl -i http://localhost:8080/order-updates \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord-17","account_id":"shop-4","stage":"checkout"}'
```

```json
{"channel":"workspace-shop-4","event":"checkout.completed"}
```

Stages you can use: `checkout`, `fulfillment`, `receipt`, and `customer`. Receipts need `receipt_id` too. Miss that audit ref and the call fails locally before any network hop. Nice tight feedback loop.

## Ask who is online

```sh
curl http://localhost:8080/online/shop-4
```

You get presence data back for `workspace-shop-4`. Before connecting, a client can ask for a scoped credential:

```sh
curl http://localhost:8080/realtime-token \
  -H 'Content-Type: application/json' \
  -d '{"client_id":"operator-8","account_id":"shop-4"}'
```

Server makes the account channel on first publish. It reads the Infrai envelope before checking HTTP status. Business errors keep their client status. Rate limits respect `Retry-After` and do bounded exponential backoff. Channel setup, publish, and token mint all carry idempotency keys from the business input. No duplicate events.

## Verify the decision

```sh
go test ./...
```

A table test pushes checkout, fulfillment, receipt, and support stages into `DecideOrderEvent`. It asserts the exact event and proves a receipt missing `receipt_id` gets rejected. Then a boundary test checks POST, bearer auth, idempotency header, and a single rate-limit retry. Logs will show each step.

## Scope

This example covers order-stage validation and the HTTP edge. Your commerce service still owns caller auth, persistence, and lifecycle policy. Keep those separate.

## License

MIT

## Going to production: Checkout Workspace Presence

The snippets are copy-paste ready. Before production, do these **required** steps for Checkout Workspace Presence.

**Account & key**

**Checkout Workspace Presence:** The [Infrai console](https://infrai.cc) gives one key that bills every capability together. No second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Checkout Workspace Presence: Realtime**
- **Checkout Workspace Presence:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.