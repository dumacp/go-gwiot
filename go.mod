module github.com/dumacp/go-gwiot

go 1.21

toolchain go1.22.0

replace github.com/nats-io/nats.go => ../../nats-io/nats.go

replace github.com/dumacp/go-schservices => ../go-schservices

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20210901041048-df2fc305778c
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dumacp/go-logs v0.0.0-20220527154118-cf272998dc98
	github.com/dumacp/keycloak v0.0.0-20191212174805-9e9a5c3da24f
	github.com/eclipse/paho.mqtt.golang v1.4.2
	github.com/gogo/protobuf v1.3.2
	github.com/google/uuid v1.5.0
	github.com/looplab/fsm v0.3.0
	github.com/nats-io/nats.go v1.22.1
	go.etcd.io/bbolt v1.3.6
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa
	golang.org/x/oauth2 v0.13.0
)

require (
	github.com/Workiva/go-datastructures v1.1.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dumacp/utils v0.0.0-20200426192206-fa29fc36dbb2 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/nats-io/nats-server/v2 v2.9.11 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/orcaman/concurrent-map v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/yanatan16/itertools v0.0.0-20160513161737-afd1891e0c4f // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
)
