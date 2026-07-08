module github.com/etclabscore/open-etc-pool

go 1.22

require (
	github.com/etclabscore/go-etchash v0.0.0-20210517131846-9a3cc202249e
	github.com/ethereum/go-ethereum v1.10.26
	github.com/gorilla/mux v1.8.0
	github.com/yvasiyarov/gorelic v0.0.7
	gopkg.in/redis.v3 v3.6.4
)

require (
	github.com/btcsuite/btcd v0.21.0-beta // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/garyburd/redigo v1.6.2 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/sys v0.16.0 // indirect
	gopkg.in/bsm/ratelimit.v1 v1.0.0-20160220154919-db14e161995a // indirect
)

replace github.com/ethereum/go-ethereum => github.com/etclabscore/core-geth v1.12.22
