# go-binance

>right now it's only the XRP and BTC pairs - more coming I swear

Beginning of a library to interact with binance


```
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
    "github.com/JakeKalstad/go-binance"
)

func main() {
	osSignals := make(chan os.Signal, 1)
	signal.Notify(
		osSignals,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGQUIT,
	)
	binanceStreamer := binance.NewBinanceStream(binance.Options{Symbol: binance.XRPBTC})
	dCh := binanceStreamer.GetDepthEventChannel()
	errCh := binanceStreamer.GetErrorChannel()
	binanceStreamer.Start()
	for {
		select {
		case evt := <-dCh:
			fmt.Printf("\nReceived a new depth event: %+v\n\n", evt)
			break
		case err := <-errCh:
			fmt.Printf("\nReceived an error %+s\n\n", err)
			panic(err)
		case sig := <-osSignals:
			fmt.Printf("Os told me to: %s", sig)
			return
		}
	}
}
```


### Enjoy