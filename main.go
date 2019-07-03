package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite/bccsp/pkcs11"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defcore"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
)

// CustomCryptoSuiteProviderFactory is will provide custom cryptosuite (bccsp.BCCSP)
type CustomCryptoSuiteProviderFactory struct {
	defcore.ProviderFactory
}

// CreateCryptoSuiteProvider returns a new default implementation of BCCSP
func (f *CustomCryptoSuiteProviderFactory) CreateCryptoSuiteProvider(config core.CryptoSuiteConfig) (core.CryptoSuite, error) {
	return pkcs11.GetSuiteByConfig(config)
}

type app struct {
	sdk      *fabsdk.FabricSDK
	profile  string
	orderer  string
	channel  string
	org      string
	username string
	peer     string
}

func newApp(profile, channel, org, username, peer string) (*app, error) {

	//sdk, err := fabsdk.New(config.FromFile(profile))
	sdk, err := fabsdk.New(config.FromFile(profile), fabsdk.WithCorePkg(&CustomCryptoSuiteProviderFactory{}))
	if err != nil {
		return nil, err
	}
	config, err := sdk.Config()
	if err != nil {
		return nil, err
	}
	if channel == "" {
		val, ok := config.Lookup("channels")
		if !ok {
			return nil, fmt.Errorf("channel is not defined in %s. Please use -channel option", profile)
		}
		channels, ok := val.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("channel is not properly defined in %s. Please use -channel option", profile)
		}
		if len(channels) < 1 {
			return nil, fmt.Errorf("channel is not defined in %s. Please use -channel option", profile)
		}
		if len(channels) > 1 {
			return nil, fmt.Errorf("multiple channels are defined in %s. Please use -channel option", profile)
		}
		for k := range channels {
			channel = k
		}
	}
	if org == "" {
		val, ok := config.Lookup("client.organization")
		if !ok {
			return nil, fmt.Errorf("client.organization is not defined in %s. Please use -org option", profile)
		}
		clientOrg, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("client.organization is not properly defined in %s. Please use -org option", profile)
		}
		org = clientOrg
	}
	if peer == "" {
		val, ok := config.Lookup("peers")
		if !ok {
			return nil, fmt.Errorf("peer is not defined in %s. Please use -peer option", profile)
		}
		peers, ok := val.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("peer is not properly defined in %s. Please use -peer option", profile)
		}
		if len(peers) < 1 {
			return nil, fmt.Errorf("peer is not defined in %s. Please use -peer option", profile)
		}
		if len(peers) > 1 {
			return nil, fmt.Errorf("multiple peers are defined in %s. Please use -peer option", profile)
		}
		for k := range peers {
			peer = k
		}
	}
	return &app{sdk: sdk, profile: profile, channel: channel, org: org, username: username, peer: peer}, nil
}

func (a *app) setOrderer(orderer string) error {
	config, err := a.sdk.Config()
	if err != nil {
		return err
	}
	if orderer == "" {
		val, ok := config.Lookup("orderers")
		if !ok {
			return fmt.Errorf("orderer is not defined in %s. Please use -orderer option", a.profile)
		}
		orderers, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("orderer is not properly defined in %s. Please use -orderer option", a.profile)
		}
		if len(orderers) < 1 {
			return fmt.Errorf("orderer is not defined in %s. Please use -orderer option", a.profile)
		}
		if len(orderers) > 1 {
			return fmt.Errorf("multiple orderers are not defined in %s. Please use -orderer option", a.profile)
		}
		for k := range orderers {
			orderer = k
		}
	}
	a.orderer = orderer
	return nil
}

func (a *app) setup() error {
	mspClient, err := mspclient.New(a.sdk.Context(), mspclient.WithOrg(a.org))
	if err != nil {
		return err
	}
	user, err := mspClient.GetSigningIdentity(a.username)
	if err != nil {
		return err
	}
	resMgmtClient, err := resmgmt.New(a.sdk.Context(fabsdk.WithUser(a.username), fabsdk.WithOrg(a.org)))
	if err != nil {
		return err
	}

	fmt.Print("Creating a channel...")
	req := resmgmt.SaveChannelRequest{
		ChannelID:         a.channel,
		ChannelConfigPath: "./channel/mychannel.tx",
		SigningIdentities: []msp.SigningIdentity{user},
	}
	_, err = resMgmtClient.SaveChannel(req, resmgmt.WithOrdererEndpoint(a.orderer))
	if err != nil {
		return err
	}
	fmt.Println(" done.")

	fmt.Print("Joining the channel...")
	err = resMgmtClient.JoinChannel(a.channel, resmgmt.WithOrdererEndpoint(a.orderer))
	if err != nil {
		return nil
	}
	fmt.Println(" done.")

	fmt.Print("Installing a chaincode...")
	ccPkg, err := packager.NewCCPackage("example", "./chaincode/go")
	if err != nil {
		return err
	}
	installCCReq := resmgmt.InstallCCRequest{
		Name:    "example",
		Path:    "example",
		Version: "v1",
		Package: ccPkg,
	}
	_, err = resMgmtClient.InstallCC(installCCReq)
	if err != nil {
		return err
	}
	fmt.Println(" done.")

	fmt.Print("Instantiating a chaincode...")
	instantiateCCReq := resmgmt.InstantiateCCRequest{
		Name:    "example",
		Path:    "example",
		Version: "v1",
		Args:    make([][]byte, 0),
		Policy:  cauthdsl.AcceptAllPolicy,
	}
	_, err = resMgmtClient.InstantiateCC(a.channel, instantiateCCReq, resmgmt.WithOrdererEndpoint(a.orderer))
	if err != nil {
		return err
	}
	fmt.Println(" done.")

	return nil
}

func (a *app) register(name string) error {
	fmt.Print("Creating a new user at CA server...")

	mspClient, err := mspclient.New(a.sdk.Context())
	if err != nil {
		return err
	}
	err = mspClient.Enroll("admin", mspclient.WithSecret("adminpw"))
	if err != nil {
		return nil
	}
	req := &mspclient.IdentityRequest{
		ID:          name,
		Affiliation: a.org,
		Type:        "client",
	}
	newIdentity, err := mspClient.CreateIdentity(req)
	if err != nil {
		return err
	}
	fmt.Println(" done.")
	fmt.Printf("\nName: %s\nSecret: %s\n", newIdentity.ID, newIdentity.Secret)
	return nil
}

func (a *app) enroll(isReenroll bool, name, secret string) error {
	if isReenroll {
		fmt.Print("Generating a pair of public/private keys, and requesting a CA server to revoke the old certificate, and create a new certificate with the public key...")
	} else {
		fmt.Print("Generating a pair of public/private keys, and sending Certificate Signing Request with the public key to a CA server...")
	}
	mspClient, err := mspclient.New(a.sdk.Context())
	if err != nil {
		return err
	}
	err = mspClient.Enroll(name, mspclient.WithSecret(secret))
	if err != nil {
		return nil
	}
	fmt.Println(" done.")
	return nil
}

func (a *app) invoke(isQuery bool, fn string, args []string) error {
	fmt.Printf("Sending a signed transaction proposal with the certificate of %s...\n", a.username)
	clientChannelContext := a.sdk.ChannelContext(a.channel, fabsdk.WithUser(a.username), fabsdk.WithOrg(a.org))
	client, err := channel.New(clientChannelContext)
	if err != nil {
		return err
	}
	var bytesArray [][]byte
	for _, arg := range args {
		bytesArray = append(bytesArray, []byte(arg))
	}
	request := channel.Request{
		ChaincodeID: "example",
		Fcn:         fn,
		Args:        bytesArray,
	}
	var response channel.Response
	if isQuery {
		response, err = client.Query(request, channel.WithTargetEndpoints(a.peer))

	} else {
		response, err = client.Execute(request, channel.WithTargetEndpoints(a.peer))
	}
	if err != nil {
		return err
	}
	res := response.Responses[0].GetResponse()
	if res.GetStatus() == 200 {
		fmt.Printf("Success\nReturned payload: %s\n", string(res.GetPayload()))
	} else {
		fmt.Println(res)
	}
	return nil
}

func main() {
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	f.Usage = func() {
		fmt.Fprintf(f.Output(), "Usage: %s [options] <command> [arguments]\n", os.Args[0])
		fmt.Fprintf(f.Output(), "Commands:\n")
		fmt.Fprintf(f.Output(), "    setup                 create/join a channel, and install/instantiate a chaincode\n")
		fmt.Fprintf(f.Output(), "    register name         register a new user\n")
		fmt.Fprintf(f.Output(), "    enroll   name secert  enroll a user\n")
		fmt.Fprintf(f.Output(), "    reenroll name secret  reenroll a user\n")
		fmt.Fprintf(f.Output(), "    execute  func args... invoke a chaincode transaction\n")
		fmt.Fprintf(f.Output(), "    query    func args... execute a chaincode query\n")
		fmt.Fprintf(f.Output(), "\nOptions:\n")
		f.PrintDefaults()
	}
	profile := f.String("profile", "connection-profile.yaml", "Connection profile")
	peer := f.String("peer", "", "Orderer name")
	orderer := f.String("orderer", "", "Orderer name")
	channel := f.String("channel", "", "Channel name")
	org := f.String("org", "", "Organization name")
	username := f.String("username", "Admin", "Username")

	f.Parse(os.Args[1:])
	args := f.Args()
	if len(args) < 1 {
		f.Usage()
		os.Exit(1)
	}

	app, err := newApp(*profile, *channel, *org, *username, *peer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	var isQuery, isReenroll bool
	switch args[0] {
	case "setup":
		app.setOrderer(*orderer)
		err = app.setup()
	case "register":
		if len(args) < 2 {
			f.Usage()
			os.Exit(1)
		}
		err = app.register(args[1])
	case "reenroll":
		isReenroll = true
		fallthrough
	case "enroll":
		if len(args) < 3 {
			f.Usage()
			os.Exit(1)
		}
		err = app.enroll(isReenroll, args[1], args[2])
	case "query":
		isQuery = true
		fallthrough
	case "execute":
		if len(args) < 2 {
			f.Usage()
			os.Exit(1)
		}
		app.setOrderer(*orderer)
		err = app.invoke(isQuery, args[1], args[2:])
	default:
		f.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
