# P2P Messenger

[![Go](https://img.shields.io/badge/Go-%2300ADD8.svg?&logo=go&logoColor=white)](#) [![Postgres](https://img.shields.io/badge/Postgres-%23316192.svg?logo=postgresql&logoColor=white)](#) [![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=fff)](#) [![Kademlia](https://img.shields.io/badge/kademlia-%23FF6600.svg?style=for-the-badge&logoColor=white)](#) [![MIT License](https://img.shields.io/badge/License-MIT-blue.svg)](#)

A decentralized peer-to-peer messenger application built in Go using Kademlia DHT for peer discovery and message routing. The application allows users to send encrypted messages directly to other peers in the network.

## Features

- Decentralized peer-to-peer architecture
- End-to-end message encryption using RSA
- Kademlia DHT for peer discovery and routing
- Docker-based deployment for easy testing
- Command-line interface for interaction
- Message history tracking
- User discovery and management

## Prerequisites

- Docker
- Docker Compose
- Make (optional, for using Makefile commands)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/yourusername/p2pmessenger.git
cd p2pmessenger
```

2. Create a `.config` directory and add initial peers (for testing through Docker):

```bash
mkdir -p src/.config
echo "172.18.0.2:1235" > src/.config/peers.txt
echo "172.18.0.3:1236" >> src/.config/peers.txt
echo "172.18.0.4:1237" >> src/.config/peers.txt
```

## Usage

### Starting the Network

Use the provided Makefile commands to manage the application:

```bash
# Build the Docker images
make build

# Start the network with three nodes
make start

# View logs from all nodes
make logs

# Stop the network
make stop

# Clean up containers and volumes
make clean
```

### Available Commands

Once connected to a node, you can use the following commands:

- `help` - Show available commands
- `send <user> <message>` - Send a message to a specific user
- `list` - List known peers and DHT state
- `history` - Show all message history
- `history <user>` - Show chat history with specific user
- `scan` - Scan for peers in the network
- `whoami` - Print current user information
- `users` - Print all registered users
- `add <address>` - Add user manually to DHT
- `exit` - Exit the application

### Example Usage

1. Connect to the first node (Adam):

```bash
docker attach messenger1
```

2. Connect to the second node (Eve) in another terminal:

```bash
docker attach messenger2
```

3. Send a message from Adam to Eve:

```bash
> send Eve Hello, how are you?
```

4. Check message history:

```bash
> history
```

## Architecture

The application uses several key components:

- **DHT (Distributed Hash Table)**: Implements Kademlia protocol for peer discovery
- **RSA Encryption**: Provides end-to-end encryption for messages
- **TCP Communication**: Handles peer-to-peer message transfer
- **Docker Network**: Simulates a distributed network environment

## Project Structure

```
P2PMessenger/
├── src/
│   ├── cmd/            # Command-line interface
│   ├── models/         # Core functionality
│   ├── types/          # Data types
│   ├── utils/          # Utility functions
│   └── main.go         # Entry point
├── scripts/            # Helper scripts
├── Dockerfile          # Container definition
├── docker-compose.yml  # Multi-container setup
└── Makefile           # Build automation
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

```
MIT License

Copyright (c) 2024 Mikhail Panteleev

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Contact
