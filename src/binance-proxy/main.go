package main

import (
	"binance-proxy/handler"
	"binance-proxy/service"
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
)

func startProxy(ctx context.Context, address string, class service.Class) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.NewHandler(ctx, class))

	log.Infof("Start %s proxy !Address: %s", class, address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("Start %s proxy failed!Error: %s", class, err)
	}
}

func handleSignal() {
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range signalChan {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			cancel()
		}
	}
}

var ctx, cancel = context.WithCancel(context.Background())
var flagSpotAddress string
var flagFuturesAddress string
var flagDeliveryAddress string 
// var flagAccountAddress string
var flagDebug bool

func main() {
	flag.StringVar(&flagSpotAddress, "s", ":8090", "spot bind address.")
	flag.StringVar(&flagFuturesAddress, "f", ":8091", "futures bind address.")
	flag.StringVar(&flagDeliveryAddress, "d", ":8092", "delivery bind address.")// 币本位合约
	// flag.StringVar(&flagAccountAddress, "a", ":8093", "account bind address.")// 账户接口
	flag.BoolVar(&flagDebug, "v", false, "print debug log.")
	flag.Parse()

	if flagDebug {
		log.SetLevel(log.DebugLevel)
	}

	go func() {
		http.ListenAndServe("0.0.0.0:65088", nil)
	}()

	go handleSignal()

	go startProxy(ctx, flagSpotAddress, service.SPOT)
	go startProxy(ctx, flagFuturesAddress, service.FUTURES)
	go startProxy(ctx, flagDeliveryAddress, service.DELIVERY)
	// go startProxy(ctx, flagAccountAddress, service.ACCOUNT)

	<-ctx.Done()

	log.Info("User interrupted..")
}
