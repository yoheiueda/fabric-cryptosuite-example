# How to use HSM from Hyperledger Fabric clients

## Install SoftHSM and OpenSC tools

```
brew install softhsm opensc
```

## Set up SoftHSM

```
softhsm2-util --init-token --slot 0 --label "ForFabric" --so-pin 1234 --pin 98765432
```

## Build the example client code

```
go get github.com/hyperledger/fabric-sdk-go
go build -o hsm-example main.go
```

## Bring up a Fabric network

```
docker-compose up -d
```

## Create a channel, and install the example chaincode

```
rm -fr ./userstore # make sure to clear old files
./hsm-example setup
```

## How to run the example client

```
./hsm-example -help
Usage: ./hsm-example [options] <command> [arguments]
Commands:
    setup                 create/join a channel, and install/instantiate a chaincode
    register name         register a new user
    enroll   name secert  enroll a user
    reenroll name secret  reenroll a user
    execute  func args... invoke a chaincode transaction
    query    func args... execute a chaincode query

Options:
  -channel string
    	Channel name
  -orderer string
    	Orderer name
  -org string
    	Organization name
  -peer string
    	Orderer name
  -profile string
    	Connection profile (default "connection-profile.yaml")
  -username string
    	Username (default "Admin")
```

## Create a new user at the CA server
```
./hsm-example register User1
```

Output should be:
```
Creating a new user at CA server... done.

Name: User1
Secret: IzsFuMAeHyaf
```
Then, enroll with the secret.
```
./hsm-example enroll User1 IzsFuMAeHyaf
```
Output should be:
```
Generating a pair of public/private keys, and sending Certificate Signing Request with the public key to a CA server... done.
```

You can check the generated certificate with the following command.
```
openssl x509 -noout -text -in userstore/hfc-kvs/User1@PeerOrg1-cert.pem
```

You can check generated keys with the following command.
```
pkcs11-tool --module /usr/local/lib/softhsm/libsofthsm2.so --list-objects --pin 98765432
```

## Run transactions with the new user
The following example calls chaincode function `put` with arguments `["abc", "123"]`.
```
./hsm-example --username User1 execute put abc 123
```
Output should be
```
Sending a signed transaction proposal with the certificate of User1...
Success
Returned payload:
```

The following example calls chaincode function `get` with an argument `["abc"]`.
```
./hsm-example -username User1 query get abc
```
Output should be:
```
Sending a signed transaction proposal with the certificate of User1...
Success
Returned payload: 123
```
