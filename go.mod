module github.com/sknr/go-coinbasepro-notifier

go 1.17

require (
	github.com/NicoNex/echotron/v3 v3.16.0
	github.com/foxever/sqlite v1.14.3
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/joho/godotenv v1.4.0
	github.com/preichenberger/go-coinbasepro/v2 v2.1.0
	github.com/recws-org/recws v1.4.0
	github.com/rs/zerolog v1.26.1
	github.com/shopspring/decimal v1.3.1
	gorm.io/gorm v1.22.4
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/tools v0.1.8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	lukechampine.com/uint128 v1.1.1 // indirect
	modernc.org/cc/v3 v3.35.22 // indirect
	modernc.org/ccgo/v3 v3.14.0 // indirect
	modernc.org/libc v1.13.2 // indirect
	modernc.org/mathutil v1.4.1 // indirect
	modernc.org/memory v1.0.5 // indirect
	modernc.org/opt v0.1.1 // indirect
	modernc.org/sqlite v1.14.3 // indirect
	modernc.org/strutil v1.1.1 // indirect
	modernc.org/token v1.0.0 // indirect
)

replace github.com/recws-org/recws v1.4.0 => github.com/sknr/recws v1.3.2-0.20211215115953-fab3c0cb58fd
