// Autogenerated by Thrift Compiler (0.9.3)
// DO NOT EDIT UNLESS YOU ARE SURE THAT YOU KNOW WHAT YOU ARE DOING

package main

import (
	"auction"
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func Usage() {
	fmt.Fprintln(os.Stderr, "Usage of ", os.Args[0], " [-h host:port] [-u url] [-f[ramed]] function [arg1 [arg2...]]:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nFunctions:")
	fmt.Fprintln(os.Stderr, "  bool bidding(i64 dealerId, i64 orderId, double price)")
	fmt.Fprintln(os.Stderr, "  bool bid(i64 dealerId, i64 orderId, double price)")
	fmt.Fprintln(os.Stderr, "  bool startAuction(Scene scenne)")
	fmt.Fprintln(os.Stderr)
	os.Exit(0)
}

func main() {
	flag.Usage = Usage
	var host string
	var port int
	var protocol string
	var urlString string
	var framed bool
	var useHttp bool
	var parsedUrl url.URL
	var trans thrift.TTransport
	_ = strconv.Atoi
	_ = math.Abs
	flag.Usage = Usage
	flag.StringVar(&host, "h", "localhost", "Specify host and port")
	flag.IntVar(&port, "p", 9090, "Specify port")
	flag.StringVar(&protocol, "P", "binary", "Specify the protocol (binary, compact, simplejson, json)")
	flag.StringVar(&urlString, "u", "", "Specify the url")
	flag.BoolVar(&framed, "framed", false, "Use framed transport")
	flag.BoolVar(&useHttp, "http", false, "Use http")
	flag.Parse()

	if len(urlString) > 0 {
		parsedUrl, err := url.Parse(urlString)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing URL: ", err)
			flag.Usage()
		}
		host = parsedUrl.Host
		useHttp = len(parsedUrl.Scheme) <= 0 || parsedUrl.Scheme == "http"
	} else if useHttp {
		_, err := url.Parse(fmt.Sprint("http://", host, ":", port))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing URL: ", err)
			flag.Usage()
		}
	}

	cmd := flag.Arg(0)
	var err error
	if useHttp {
		trans, err = thrift.NewTHttpClient(parsedUrl.String())
	} else {
		portStr := fmt.Sprint(port)
		if strings.Contains(host, ":") {
			host, portStr, err = net.SplitHostPort(host)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error with host:", err)
				os.Exit(1)
			}
		}
		trans, err = thrift.NewTSocket(net.JoinHostPort(host, portStr))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error resolving address:", err)
			os.Exit(1)
		}
		if framed {
			trans = thrift.NewTFramedTransport(trans)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating transport", err)
		os.Exit(1)
	}
	defer trans.Close()
	var protocolFactory thrift.TProtocolFactory
	switch protocol {
	case "compact":
		protocolFactory = thrift.NewTCompactProtocolFactory()
		break
	case "simplejson":
		protocolFactory = thrift.NewTSimpleJSONProtocolFactory()
		break
	case "json":
		protocolFactory = thrift.NewTJSONProtocolFactory()
		break
	case "binary", "":
		protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
		break
	default:
		fmt.Fprintln(os.Stderr, "Invalid protocol specified: ", protocol)
		Usage()
		os.Exit(1)
	}
	client := auction.NewAuctionServiceClientFactory(trans, protocolFactory)
	if err := trans.Open(); err != nil {
		fmt.Fprintln(os.Stderr, "Error opening socket to ", host, ":", port, " ", err)
		os.Exit(1)
	}

	switch cmd {
	case "bidding":
		if flag.NArg()-1 != 3 {
			fmt.Fprintln(os.Stderr, "Bidding requires 3 args")
			flag.Usage()
		}
		argvalue0, err8 := (strconv.ParseInt(flag.Arg(1), 10, 64))
		if err8 != nil {
			Usage()
			return
		}
		value0 := argvalue0
		argvalue1, err9 := (strconv.ParseInt(flag.Arg(2), 10, 64))
		if err9 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		argvalue2, err10 := (strconv.ParseFloat(flag.Arg(3), 64))
		if err10 != nil {
			Usage()
			return
		}
		value2 := argvalue2
		fmt.Print(client.Bidding(value0, value1, value2))
		fmt.Print("\n")
		break
	case "bid":
		if flag.NArg()-1 != 3 {
			fmt.Fprintln(os.Stderr, "Bid requires 3 args")
			flag.Usage()
		}
		argvalue0, err11 := (strconv.ParseInt(flag.Arg(1), 10, 64))
		if err11 != nil {
			Usage()
			return
		}
		value0 := argvalue0
		argvalue1, err12 := (strconv.ParseInt(flag.Arg(2), 10, 64))
		if err12 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		argvalue2, err13 := (strconv.ParseFloat(flag.Arg(3), 64))
		if err13 != nil {
			Usage()
			return
		}
		value2 := argvalue2
		fmt.Print(client.Bid(value0, value1, value2))
		fmt.Print("\n")
		break
	case "startAuction":
		if flag.NArg()-1 != 1 {
			fmt.Fprintln(os.Stderr, "StartAuction requires 1 args")
			flag.Usage()
		}
		arg14 := flag.Arg(1)
		mbTrans15 := thrift.NewTMemoryBufferLen(len(arg14))
		defer mbTrans15.Close()
		_, err16 := mbTrans15.WriteString(arg14)
		if err16 != nil {
			Usage()
			return
		}
		factory17 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt18 := factory17.GetProtocol(mbTrans15)
		argvalue0 := auction.NewScene()
		err19 := argvalue0.Read(jsProt18)
		if err19 != nil {
			Usage()
			return
		}
		value0 := argvalue0
		fmt.Print(client.StartAuction(value0))
		fmt.Print("\n")
		break
	case "":
		Usage()
		break
	default:
		fmt.Fprintln(os.Stderr, "Invalid function ", cmd)
	}
}
