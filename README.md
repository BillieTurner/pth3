# pth3
A Censorship-Resistant Tor Pluggable Transport

# How to build:
```
make build
```

or

```
cd pth3/src/client
go build
```


# Client `torrc` Config
```
UseBridges 1
Bridge pth3 <Server IP>:<Server Port>

ClientTransportPlugin pth3 exec path/to/pth3-client -log-file path/to/pth3-client.log -public-key-pin <public key fingerprint>
```

# Server `torrc` Config
```
BridgeRelay 1
ORPort 9001
ExtORPort 9002

ServerTransportPlugin pth3 exec path/to/pth3-server -log-file path/to/pth3-server.log -certificate cert.pem -key key.pem
ServerTransportListenAddr pth3 0.0.0.0:<Port>
```

# Create a private-public key pair
```
openssl genrsa -out private.key 2048
openssl req -new -x509 -key private.key -out publickey.cer -days 365
```

# Get a public key fingerprint
```
openssl x509 -in publickey.cer -fingerprint -sha256 -noout | tr A-Z a-z | tr -d :
```