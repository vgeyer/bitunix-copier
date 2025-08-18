# Bitunix Order Copier MVP

A simple Go application that monitors orders on a source Bitunix account via websocket and copies them to a destination account using REST API.

## Features

- Real-time order monitoring via websocket
- Automatic order copying between accounts
- Environment-based configuration
- Graceful shutdown handling
- Error logging and recovery

## Setup

1. Set environment variables:
```bash
export SOURCE_API_KEY="your_source_api_key"
export SOURCE_SECRET_KEY="your_source_secret_key"
export DEST_API_KEY="your_destination_api_key"
export DEST_SECRET_KEY="your_destination_secret_key"
```

2. Build and run:
```bash
go build -o bitunix-copier
./bitunix-copier
```

## How it works

1. **Monitoring**: Connects to source account websocket and subscribes to order events
2. **Filtering**: Only processes NEW and PARTIALLY_FILLED orders
3. **Copying**: Places identical orders on the destination account via REST API
4. **Logging**: Provides detailed logs of all operations

## Safety Notes

- Test with small amounts first
- Monitor both accounts during operation
- Ensure sufficient balance on destination account
- Consider rate limits and latency