package hostpath

import (
    // "github.com/container-storage-interface/spec/lib/go"

	"github.com/thecodeteam/gocsi"
	"os"
	"net"
	"fmt"
	"regexp"
	"errors"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"flag"
)

const(
	// Environment Variable for CSI Endpoint
	CSIEndpoint = "CSI_ENDPOINT"
)

const (
	protoAddrGuessPatt = `(?i)^(?:tcp|udp|ip|unix)[^:]*://`

	protoAddrExactPatt = `(?i)^((?:(?:tcp|udp|ip)[46]?)|` +
		`(?:unix(?:gram|packet)?))://(.+)$`
)

var (
	emptyRX          = regexp.MustCompile(`^\s*$`)
	protoAddrGuessRX = regexp.MustCompile(protoAddrGuessPatt)
	protoAddrExactRX = regexp.MustCompile(protoAddrExactPatt)
)

func GetCSIEndpoint() (network, addr string, err error) {
	protoAddr := os.Getenv(CSIEndpoint)
	if emptyRX.MatchString(protoAddr) {
		return "", "",  errors.New("missing CSI_ENDPOINT")
	}
	return ParseProtoAddr(protoAddr)
}

// ParseProtoAddr parses a Golang network address.
func ParseProtoAddr(protoAddr string) (proto string, addr string, err error) {

	if emptyRX.MatchString(protoAddr) {
		return "", "",  errors.New("non-empty network address is required")
	}

	// If the provided network address does not begin with one
	// of the valid network protocols then treat the string as a
	// file path.
	//
	// First check to see if the file exists at the specified path.
	// If it does then assume it's a valid file path and return it.
	//
	// Otherwise attempt to create the file. If the file can be created
	// without error then remove the file and return the result a UNIX
	// socket file path.
	if !protoAddrGuessRX.MatchString(protoAddr) {

		// If the file already exists then assume it's a valid sock
		// file and return it.
		if _, err := os.Stat(protoAddr); !os.IsNotExist(err) {
			return "unix", protoAddr, nil
		}

		f, err := os.Create(protoAddr)
		if err != nil {
			return "", "", fmt.Errorf(
				"invalid implied sock file: %s: %v", protoAddr, err)
		}
		if err := f.Close(); err != nil {
			return "", "", fmt.Errorf(
				"failed to verify network address as sock file: %s", protoAddr)
		}
		if err := os.RemoveAll(protoAddr); err != nil {
			return "", "", fmt.Errorf(
				"failed to remove verified sock file: %s", protoAddr)
		}
		return "unix", protoAddr, nil
	}

	// Parse the provided network address into the protocol and address parts.
	m := protoAddrExactRX.FindStringSubmatch(protoAddr)
	if m == nil {
		return "", "", fmt.Errorf("invalid network address: %s", protoAddr)
	}
	return m[1], m[2], nil
}


// GetCSIEndpoint returns the network address specified by the
// environment variable CSI_ENDPOINT.
func GetCSIEndpointListener() (net.Listener, error) {
	proto, addr, err := GetCSIEndpoint()
	if err != nil {
		return nil, err
	}
	return net.Listen(proto, addr)
}


func main() {
	listen, err := gocsi.GetCSIEndpointListener()
	if err != nil {
		glog.Fatalf("failed to listen: %v\n", err)
	}

	// Define a lambda that can be used in the exit handler
	// to remove a potential UNIX sock file.


	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	//pb.RegisterRouteGuideServer(grpcServer, newServer())
	grpcServer.Serve(listen)
}
