# Real DNS Testing

Local tests use a fake domain such as `test.com` and send UDP packets directly
to a local Slipstream DNS listener. That proves the protocol and tunnel, but it
does not test public DNS delegation.

There are two useful real-world modes.

## Direct Resolver Mode

Generate a test certificate:

```sh
slipstream-cert -hosts tunnel.example.com,VPS_IP -cert server.crt -key server.key
```

Run the server on a VPS and send the client directly to that VPS:

```sh
slipstream-server -dns :53 -domain tunnel.example.com -cert server.crt -key server.key
slipstream-client -resolver VPS_IP:53 -domain tunnel.example.com -cert-fingerprint PRINTED_SHA256
```

In this mode the domain only needs to match the domain configured in the
Slipstream server. Public DNS records are not required because the client sends
queries directly to `VPS_IP:53`.

## Public Resolver Mode

To test through normal recursive DNS resolvers, delegate a subdomain to your
Slipstream server.

Example zone records:

```text
tunnel.example.com.      NS  ns1.tunnel.example.com.
ns1.tunnel.example.com.  A   VPS_IP
```

Then run:

```sh
slipstream-server -dns :53 -domain tunnel.example.com -cert server.crt -key server.key
slipstream-client -resolver 1.1.1.1:53 -domain tunnel.example.com -cert-fingerprint PRINTED_SHA256
```

The recursive resolver receives queries under `tunnel.example.com`, follows the
NS delegation, and sends them to the Slipstream server.

## Notes

- UDP port 53 must be reachable from the client or recursive resolver.
- Some networks block or rewrite DNS traffic; direct resolver mode is the
  simplest first VPS smoke test.
- Use certificate pinning for production-like tests.
- `-allow-insecure` is only for local bring-up.
