module github.com/bnb-chain/zkbnb

go 1.16

require (
	github.com/apolloconfig/agollo/v4 v4.2.1
	github.com/bnb-chain/zkbnb-go-sdk v1.0.4-0.20221012063144-3a6e84095b4d
	github.com/dgraph-io/ristretto v0.1.0
	github.com/gin-gonic/gin v1.8.1
	github.com/hashicorp/golang-lru v0.5.5-0.20221011183528-d4900dc688bf
	github.com/ory/dockertest v3.3.5+incompatible
	github.com/panjf2000/ants/v2 v2.5.0
	github.com/prometheus/client_golang v1.14.0
	github.com/swaggo/files v0.0.0-20220728132757-551d4a08d97a
	github.com/swaggo/gin-swagger v1.5.3
	github.com/swaggo/swag v1.8.8
	github.com/zeromicro/go-zero v1.3.4
	gorm.io/gorm v1.24.0
	gorm.io/plugin/dbresolver v1.3.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/go-openapi/spec v0.20.7 // indirect
	github.com/go-openapi/swag v0.22.3
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/klauspost/cpuid/v2 v2.2.2 // indirect
	github.com/libp2p/go-libp2p v0.24.1 // indirect
	github.com/libp2p/go-libp2p-core v0.20.1 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/runc v1.1.4 // indirect
	github.com/whyrusleeping/tar-utils v0.0.0-20201201191210-20a61371de5b // indirect
	golang.org/x/crypto v0.4.0 // indirect
	golang.org/x/net v0.4.0 // indirect
	golang.org/x/tools v0.4.0 // indirect
)

require (
	github.com/bnb-chain/zkbnb-crypto v0.0.8-0.20221222075728-240d4c7279b7
	github.com/bnb-chain/zkbnb-eth-rpc v0.0.2
	github.com/bnb-chain/zkbnb-smt v0.0.3-0.20221118180206-7685632073d8
	github.com/consensys/gnark v0.7.0
	github.com/consensys/gnark-crypto v0.7.0
	github.com/eko/gocache/v2 v2.3.1
	github.com/ethereum/go-ethereum v1.10.26
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-jwt/jwt/v4 v4.4.2 // indirect
	github.com/ipfs-cluster/ipfs-cluster v1.0.4 // indirect
	github.com/ipfs/go-ipfs-api v0.3.0
	github.com/ipfs/go-ipfs-files v0.1.1
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mr-tron/base58 v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/cors v1.8.2 // indirect
	github.com/stretchr/testify v1.8.1
	github.com/urfave/cli v1.22.10 // indirect
	github.com/urfave/cli/v2 v2.23.6
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	gorm.io/driver/postgres v1.3.6
	k8s.io/apimachinery v0.24.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220328201542-3ee0da9b0b42
)

replace (
	github.com/consensys/gnark => github.com/bnb-chain/gnark v0.7.1-0.20221031143243-a94d59b60efe
	github.com/consensys/gnark-crypto => github.com/bnb-chain/gnark-crypto v0.7.1-0.20221115030433-6e0195f27b89

)

replace github.com/bnb-chain/zkbnb-smt => github.com/qct/zkbnb-smt v0.0.0-20221203161605-59c0f417b4e8

replace github.com/bnb-chain/zkbnb-crypto => /Users/user/zk/fork/ipfs/zkbnb-crypto
