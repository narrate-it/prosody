# SES verified-event path — Go/Pulumi

Declares the consumer-owned AWS event path required by the AMOS verified-event
contract (`delivery/email/ses.NewSNSVerifiedSource`): a dedicated SNS topic,
an SQS subscription that preserves native SNS signatures, a dead-letter queue
for review, and the SES `SEND` event destination on the existing
`prosody-identity` configuration set. The host-side composition
(`backend/seslink`) consumes this path: SQS ingestion authenticates every
envelope with trusted signing-cert pins before insert-only PostgreSQL
retention (`backend/seslink/schema.sql`), and AMOS reauthenticates retained
bytes on every reconciliation.

This program creates no IAM credentials, sends no mail and changes no DNS.
The sibling `infra/ses` stack owns the identity/DKIM/configuration set; this
stack owns only the event path. Both pin AWS provider v7.54.0, require the
explicit account/region selection, refuse the root principal, and protect +
retain every resource.

## Owner gate: IAM extension (blocking)

The runtime/IaC profile `prosody-ses` carries the SES-only policy
`prosody-ses-boundary` (exact document: `infra/ses/iam-policy.json`). Verified
live denials (2026-11-13, us-west-2): `sns:CreateTopic`, `sqs:CreateQueue`,
`ses:CreateConfigurationSetEventDestination`, `iam:ListPolicies`. The profile
cannot extend itself; the owner's console login (`aws login`) is the admin
path and its cached session has expired.

`iam-policy-events.json` is the proposed **full replacement** document: the
existing SES statements unchanged, plus bounded SNS/SQS management on the two
exact queue ARNs and the exact topic ARN, SES event-destination management on
the exact configuration-set ARN, list actions region-conditioned, and runtime
`sqs:ReceiveMessage`/`sqs:DeleteMessage` on the same queues. An admin
principal applies it with:

```sh
aws iam create-policy-version \
  --policy-arn arn:aws:iam::394824061124:policy/prosody-ses-boundary \
  --policy-document file://iam-policy-events.json --set-as-default
```

No other principal or resource is admitted. Deletion actions remain absent.

## Apply and qualification (after the IAM gate)

1. `AWS_PROFILE=prosody-ses pulumi preview --diff` then a reviewed
   `pulumi up` (stack `prod`). Same-account SQS subscriptions auto-confirm;
   no public endpoint is required.
2. Acquire the regional SNS signing certificate independently: publish one
   test notification to the topic (admin or `sns:Publish` grant), read its
   `SigningCertURL` from the retained queue message, fetch that exact URL over
   verified HTTPS (`sns.us-west-2.amazonaws.com`), validate provenance, record
   the SHA-256 fingerprint, and retain the PEM. Pins are privileged trust
   configuration; rotate explicitly with at most two overlapping valid pins.
3. Write operator custody `ses.json` (private, never committed): region,
   accountId, topicArn, identityArn
   (`arn:aws:ses:us-west-2:<account>:identity/prosody.chat`), configurationSet,
   credentialProfile, queueUrl, and `trustedCertificates`
   (`signingCertUrl` + `certificatePem`).
4. Apply `backend/seslink/schema.sql` with owner DB custody (already applied
   to `prosody-db` on 2026-11-13).
5. Qualify the real journey: `PROSODY_PRIVATE=<custody> go run
   ./backend/sesqualify -recipient <verified address>` sends one real SES
   sign-in email, ingests the signed Send evidence and reconciles to an
   accepted receipt. Sandbox accounts may only send to verified recipients;
   production sending access is a separate AWS account gate.

## Budget

SNS and SQS at this volume are within the AWS free tier (1M requests/month);
SES is $0.10 per 1,000 emails. No compute is added by this stack.

## Verification scope

Mock registration tests check the protected graph, the bounded source
policies, native-signature subscription, SEND-only matching, account/root
gates and the 24h-aligned retention. They do not contact AWS. Live preview,
apply, pin provenance and the real send/reconcile journey remain separate
operating gates; none is claimed by source tests.
