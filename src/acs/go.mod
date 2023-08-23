module bkr-go/acs

go 1.16

require (
	github.com/perlin-network/noise v1.1.3 // indirect
	go.dedis.ch/kyber/v3 v3.0.13 // indirect
	bkr-go/crypto v0.0.0-00010101000000-000000000000
	bkr-go/transport v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.16.0
	google.golang.org/protobuf v1.30.0 // indirect
)

replace bkr-go/transport => ../transport

replace bkr-go/crypto => ../crypto
