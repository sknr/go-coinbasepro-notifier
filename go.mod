module github.com/sknr/go-coinbasepro-notifier

go 1.16

require (
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/gorilla/websocket v1.4.2
	github.com/jinzhu/now v1.1.2 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/preichenberger/go-coinbasepro/v2 v2.0.6-0.20210403140934-f2b4b86ec877
	github.com/rs/zerolog v1.20.0
	github.com/shopspring/decimal v1.2.0
	github.com/yanzay/tbot/v2 v2.2.0
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.3
)

replace github.com/yanzay/tbot/v2 => github.com/sknr/tbot/v2 v2.2.1-0.20210322202531-c00af010167e
