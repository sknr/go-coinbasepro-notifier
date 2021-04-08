#Coinbase Pro Notifier

The intention of the app is to inform the Coinbase Pro user about oder changes via telegram messages

### Usage
1. In order to properly use the app, copy the `.env_example` to `.env` and provide the necessary Coinbase Pro API-Key and your telegram bot details.
2. Run `go run cmd/notfier.go`

### Raspberry Pi usage
Use a raspberry Pi with docker support e.g. [HypriotOS](https://blog.hypriot.com/downloads/)

1. In order to properly use the app, copy the `.env_example` to `.env` and provide the necessary Coinbase Pro API-Key and your telegram bot details.
2. Run `make install-pi`
3. After connecting to the raspberry pi, `cd app` and run the local build docker image from the previous step with: `docker-compose up -d`
4. Place and/or remove some orders within your Coinbase Pro account, and you'll receive order updates via telegram messages.


