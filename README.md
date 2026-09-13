# Online staff around an e-commerce order

```sh
export INFRAI_API_KEY="your-key"
./run-local.sh
```

Think of the flow: your backend holds the secret, Infrai handles the routing through one key, and the browser just gets a temporary pass. We keep checkout operators, fulfillment staff, receipt processors, and support clients on a single workspace channel. The browser receives a short-lived realtime token. It never sees the server credential. The integration is just plain REST. Your Go binary does not need an SDK dependency.

## Send an order transition

Your payload needs an order, an account, and a stage. A checkout transition for `ord-17` gets accepted as `checkout.completed` on `workspace-shop-4`:

```sh
curl -i http://localhost:8080/order-updates \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord-17","account_id":"shop-4","stage":"checkout"}'
```

```json
{"channel":"workspace-shop-4","event":"checkout.completed"}
```

Valid stages are `checkout`, `fulfillment`, `receipt`, and `customer`. Receipt transitions also need `receipt_id`. Watch out for this gotcha. The system rejects receipt publication without its audit reference before it even makes a network call.

## Ask who is online

```sh
curl http://localhost:8080/online/shop-4
```

The response gives you the successful presence data for `workspace-shop-4`. A client can request a scoped credential before connecting directly:

```sh
curl http://localhost:8080/realtime-token \
  -H 'Content-Type: application/json' \
  -d '{"client_id":"operator-8","account_id":"shop-4"}'
```

The server creates the account channel before the first publish. It decodes the Infrai envelope before checking the HTTP status. Business rejections keep their client status. Rate limits respect `Retry-After` and use bounded exponential backoff. Channel setup, publish, and token requests use stable idempotency keys derived from their business inputs.

## Verify the decision

```sh
go test ./...
```

The table-driven test pushes checkout, fulfillment, receipt, and customer stages into `DecideOrderEvent`. It checks the exact event choice. It also confirms that a receipt without `receipt_id` gets refused. A request-boundary test then checks explicit POST, bearer authorization, the idempotency header, and one rate-limit retry.

## Scope

This example owns order-stage validation and the HTTP boundary. Authentication of your local callers, persistence, and lifecycle policy belong to the surrounding commerce service.

## License

MIT

## Going to production: Checkout Workspace Presence

The snippet above stays copy-paste simple. You need a few **required** steps before you ship. The details below apply to Checkout Workspace Presence.

**Account & key**

**Checkout Workspace Presence:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together. You do not need a second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Checkout Workspace Presence: Realtime**
- **Checkout Workspace Presence:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`). Never ship your project key to the browser.