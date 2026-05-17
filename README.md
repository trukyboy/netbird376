# Netbird-376

Open source self-hosted VPN platform based on [Netbird](https://github.com/netbirdio/netbird).

## Features

- All Netbird features
- Cross-network peer sharing
- Automatic subdomain provisioning via vp376.net
- Auto TLS certificates
- Multi-tenant architecture

## Quick Install

    curl -fsSL https://vp376.net/install.sh | bash

## Architecture

    vp376.net (Master)
    └── dj5lgk.vp376.net (Client A)
    └── x9k2mp.vp376.net (Client B)
    └── r4tz8q.vp376.net (Client C)

## License

BSD-3-Clause
