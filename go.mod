module github.com/etclabscore/open-etc-pool

go 1.24

require (
	github.com/etclabscore/go-etchash v0.0.0-20210517131846-9a3cc202249e
	github.com/ethereum/go-ethereum v1.10.26
	github.com/gorilla/mux v1.8.0
	github.com/prometheus/client_golang v1.12.0
	github.com/redis/go-redis/v9 v9.21.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcd v0.21.0-beta // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/prometheus/client_model v0.2.1-0.20210607210712-147c58e9608a // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/sys v0.30.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/ethereum/go-ethereum => github.com/etclabscore/core-geth v1.12.22
