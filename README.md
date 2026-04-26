# Solana Slot Latency Tracker (Go)

This repository measures the latency between when a slot is first observed via **Solana gossip** and when it is confirmed via a **Yellowstone gRPC** stream.

## What It Does

- Connects to Solana gossip and subscribes to vote messages.
- Simultaneously subscribes to slot updates from a Yellowstone gRPC endpoint.
- For each slot, records the first arrival timestamp from each source.
- When both timestamps are available, logs the latency (`gossip arrival → Yellowstone slot update`).
- Writes structured JSON logs to stdout and a file via zerolog.

## Repository Layout

- `runtime/`: executable entrypoint (`main.go`).
- `services/`: `TrackerService` — coordinates gossip and Yellowstone subscriptions, slot tracking logic.
- `.env.example`: environment variable template.

## Requirements

- Go `1.25.5` (as declared in `go.mod`).
- A Yellowstone-compatible gRPC endpoint and access token.
- UDP and TCP ports open for gossip traffic.

## Configuration

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

| Variable        | Description                                      | Example                        |
|-----------------|--------------------------------------------------|--------------------------------|
| `ENV`           | Solana cluster (`mainnet`, `testnet`, `devnet`)  | `mainnet`                      |
| `GRPC_ENDPOINT` | Yellowstone gRPC endpoint host                   | `yellowstone.eu.fluxrpc.com`   |
| `GRPC_TOKEN`    | Auth token for the gRPC endpoint                 | `432d443f-...`                 |
| `UDP_PORT`      | UDP port for gossip traffic                      | `8001`                         |
| `TCP_PORT`      | TCP port for gossip IP echo handshake            | `8001`                         |
| `LOG_LEVEL`     | Log verbosity (`trace`, `debug`, `info`, `error`)| `info`                         |
| `LOG_FILE`      | Path to write the log file                       | `gossip.log`                   |
| `DISPLAY_SLOTS` | Whether to display slot output                   | `true`                         |

## Quick Start

```bash
go run ./runtime/main.go
```

On first start, if `id.json` is not found, a new keypair is generated and written there automatically.

## How Latency Is Measured

The tracker keeps a circular ring buffer of the last `1500` slots. For each slot:

1. When a **gossip vote** for that slot arrives, the receive timestamp is stored.
2. When a **Yellowstone slot update** for the same slot arrives, the receive timestamp is stored.
3. Once both timestamps are present, the latency is computed as:

```
latency = yellowstone_arrival − gossip_arrival
```

The result is a `time.Duration` and is logged as the `latency` field on the `SLOT LATENCY TRACKED` log line.

## Wire Compatibility Note

The gossip dependency (`github.com/Gealber/gossip`) relies on a forked binary decoder
(`github.com/Gealber/binary`) to correctly handle Solana's bincode wire format. Because Go only
applies `replace` directives from the main module, this repo must redeclare the replacement in its
own `go.mod`:

```
replace github.com/gagliardetto/binary => github.com/Gealber/binary v0.0.0-20260419181258-89e8fe42e32c
```

Without this, gossip packet decoding will fail with `unknown CrdsData discriminant` errors caused
by byte-offset drift in the upstream library.

## Disclaimer

This is a narrow observability utility focused on measuring gossip-to-confirmation latency on
Solana. It is not a full gossip node and does not persist or re-broadcast any state.
