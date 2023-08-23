module bkr-go/client

go 1.16

require (
	bkr-go/crypto v0.0.0-00010101000000-000000000000 // indirect
	bkr-go/transport v0.0.0-00010101000000-000000000000 // indirect
	google.golang.org/protobuf v1.30.0
)

replace bkr-go/transport => ../transport

replace bkr-go/crypto => ../crypto
