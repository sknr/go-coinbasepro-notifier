# Coinbase Pro Notifier

The intention of this app is to inform the Coinbase Pro user about order changes via telegram messages. Therefore, the
app exposes a webserver where the users can log in via [Telegram Login](https://core.telegram.org/widgets/login), and
set up the Coinbase Pro API-Key.

Here's an example of how the app works:

1. The Telegram-Bot: https://t.me/CoinbaseProNotifierBot
2. The Webserver: https://notifier.bot.apperia.de

> HINT: If you don't already have an Telegram bot, have a look at https://core.telegram.org/bots.
---

### Setup Coinbase Pro API-Key

Official Coinbase Pro help:

- https://help.coinbase.com/en/pro/other-topics/api/how-do-i-create-an-api-key-for-coinbase-pro

Example of API-Key creation with images:

- https://cryptopro.app/help/automatic-import/coinbase-pro-api-key/

> HINT: The only required API-Key permission is "view".

---

### Usage
1. In order to properly use the app, copy the `.env_example` to `.env` and provide the necessary Coinbase Pro API-Key and your telegram bot details.
2. Run `go run cmd/notfier.go`

### Usage with docker

1. Run `docker build -t coinbasepro-notifier .`
2. Run `docker run --rm -p 8080:8080 -v "$(pwd)/data:/app/data" --env-file .env coinbasepro-notifier`
3. Press `CTRL+C` to shut down the server.

### Usage with docker-compose

1. Customize the example `docker-compose.yml file to your needs.
1. Run `docker-compose up -d` to start the service.

> HINT: For a quick try, use the prebuild docker image from docker-hub: [coinbasepro-notifier](https://hub.docker.com/r/sknr/coinbasepro-notifier)
