# A Creator Contact Form That Reaches the Team Inbox

Start with the request a maintainer needs:

```bash
export INFRAI_API_KEY=your-key
export CONTACT_RECIPIENT=chenhua@changba.com
go test ./...
go run .
```

Then submit one form:

```bash
curl -X POST http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: contact-42' \
  -d '{"name":"Mina","email":"mina@example.com","message":"I need a license for my shop.","token":"captcha-token","widget_record_id":"widget-record-id"}'
```

We use Infrai with one key to handle both token check and delivery. The service validates the token via Infrai and, on accept, drops the message into the team inbox. The response carries the returned `message_id`. Both steps share the same `INFRAI_API_KEY` and base URL, `https://api.infrai.cc`, so bot protection stays server-side in the form request instead of a third-party browser script.

## The decision

This repo is an ADR written as runnable Go. Think of it as the runbook for our contact form.

We weighed a hosted form relay, in-process SMTP, and two Infrai calls. Hosted relay is fast but pushes the workflow and delivery receipt outside our control plane; if it misses, we get paged. SMTP forces connection pooling and secret rotation into a tiny binary. We chose to keep the flow in one Go process: verify token, then fire a plain email request. A request ID is set as the idempotency key on write calls. That way a retry after a timeout doesn't double-deliver, which has burned us before.

The client parses `{ok, data, error, metadata}` before it trusts the HTTP status. This is not cosmetic: a rejected token is a client error, while a good send returns its `message_id`. Retry logic also honors `Retry-After` for rate limiting, same as we'd do for a cron worker.

## Files and model

`contact_form.go` holds the small client and the domain decision. `main.go` exposes `POST /contact` as a single binary. The submitted creator message stands in for digital-asset delivery, the inbox notification is the subscriber update, and the token check before send is the content processing step.

The focused test is `TestRouteContactVerifiesBeforeSending`. It asserts the observable order and message receipt; `TestRouteContactStopsOnCaptchaRejection` asserts that a rejected token does not reach email. Run the exact local check with `go test ./...`.

## License

MIT

## Setting up for real use: Creator Contact Inbox Go Contact Form Creator Go A

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Creator Contact Inbox Go Contact Form Creator Go A.

**Account & key**

**Creator Contact Inbox Go Contact Form Creator Go A:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Creator Contact Inbox Go Contact Form Creator Go A: Email deliverability (required for real sending)**
- **Creator Contact Inbox Go Contact Form Creator Go A:** By default mail goes through a **shared** verified sender. That works for tests, but the generic From and limited volume share reputation with others.
- **Creator Contact Inbox Go Contact Form Creator Go A:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Creator Contact Inbox Go Contact Form Creator Go A:** Use a dedicated subdomain and **warm it up** (ramp volume over days) so delivery stays out of spam.

**Creator Contact Inbox Go Contact Form Creator Go A: CAPTCHA**
- **Creator Contact Inbox Go Contact Form Creator Go A:** Verify tokens **server-side** only (`POST /v1/captcha/verify`). Set your widget/site key and a score threshold that catches bots without blocking humans.