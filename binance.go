package binance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/websocket"
)

const (
	XRPBTC = "xrpbtc"
)

type Options struct {
	Symbol        string
	LookbackLimit int
}

func NewBinanceStream(opts Options) BinanceStream {
	return &binanceStream{
		Options:           opts,
		LastUpdateID:      -1,
		DepthEventChannel: make(chan DepthEvent),
		ErrorChannel:      make(chan error),
	}
}

type binanceStream struct {
	Options           Options
	LastUpdateID      int64
	DepthEventChannel chan DepthEvent
	ErrorChannel      chan error
}

type BinanceStream interface {
	Start()
	GetDepthEventChannel() chan DepthEvent
	GetErrorChannel() chan error
}

type BidCount struct {
	Price  string
	Volume string
}

type Bids []*BidCount

type DepthEvent struct {
	EventType     string
	EventTime     int64
	Symbol        string
	FirstUpdateID int64
	FinalUpdateID int64
	BidsToUpdate  Bids
	AsksToUpdate  Bids
}

type rawBids [][]interface{}
type depth struct {
	LastUpdateID int64   `json:"lastUpdateId"`
	Bids         rawBids `json:"bids"`
}

type depthEvent struct {
	EventType     string  `json:"e"`
	EventTime     int64   `json:"E"`
	Symbol        string  `json:"s"`
	FirstUpdateID int64   `json:"U"`
	FinalUpdateID int64   `json:"u"`
	BidsToUpdate  rawBids `json:"b"`
	AsksToUpdate  rawBids `json:"a"`
}

func (b *binanceStream) GetDepthEventChannel() chan DepthEvent {
	return b.DepthEventChannel
}

func (b *binanceStream) GetErrorChannel() chan error {
	return b.ErrorChannel
}

func (b *binanceStream) Start() {
	go func() {
		e := b.getDepth(b.Options.Symbol, 1000)
		if e != nil {
			b.ErrorChannel <- e
		}
		b.processDepthEvents()
	}()
}

func transformRawBids(rawBids rawBids) Bids {
	bids := Bids{}
	for _, bSet := range rawBids {
		price := ""
		vol := ""
		for idx, rBid := range bSet {
			val, ok := rBid.(string)
			if ok {
				if idx == 0 {
					price = val
				}
				if idx == 1 {
					vol = val
				}
			}
		}
		bids = append(bids, &BidCount{
			Price:  price,
			Volume: vol,
		})
	}
	return bids
}

func (b *binanceStream) processTradeEvents() {
	origin := "http://localhost/"
	url := b.getWebsocketURL("trade")
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		b.ErrorChannel <- err
		return
	}
	if _, err := ws.Write(nil); err != nil {
		b.ErrorChannel <- err
		return
	}
	var msg = make([]byte, 4096)
	var n int
	n, err = ws.Read(msg)
	if err != nil {
		b.ErrorChannel <- err
		return
	}
	for err == nil {
		evt := depthEvent{}
		e := json.Unmarshal(msg[:n], &evt)
		if e != nil {
			b.ErrorChannel <- e
			return
		}
		if evt.FinalUpdateID > b.LastUpdateID {
			b.DepthEventChannel <- DepthEvent{
				EventType:     evt.EventType,
				EventTime:     evt.EventTime,
				Symbol:        evt.Symbol,
				FirstUpdateID: evt.FirstUpdateID,
				FinalUpdateID: evt.FinalUpdateID,
				BidsToUpdate:  transformRawBids(evt.BidsToUpdate),
				AsksToUpdate:  transformRawBids(evt.AsksToUpdate),
			}
		}
		n, err = ws.Read(msg)
	}
}

func (b *binanceStream) getWebsocketURL(view string) string {
	return fmt.Sprintf("wss://stream.binance.com:9443/ws/%s@%s", b.Options.Symbol, view)
}

func (b *binanceStream) processDepthEvents() {
	origin := "http://localhost/"
	url := b.getWebsocketURL("depth")
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		fmt.Println(err)
		b.ErrorChannel <- err
		return
	}
	if _, err := ws.Write(nil); err != nil {
		b.ErrorChannel <- err
		return
	}
	var msg = make([]byte, 4096)
	var n int
	n, err = ws.Read(msg)
	if err != nil {
		b.ErrorChannel <- err
		return
	}
	for err == nil {
		evt := depthEvent{}
		e := json.Unmarshal(msg[:n], &evt)
		if e != nil {
			fmt.Println(msg)
			b.ErrorChannel <- e
			return
		}
		if evt.FinalUpdateID > b.LastUpdateID {
			b.DepthEventChannel <- DepthEvent{
				EventType:     evt.EventType,
				EventTime:     evt.EventTime,
				Symbol:        evt.Symbol,
				FirstUpdateID: evt.FirstUpdateID,
				FinalUpdateID: evt.FinalUpdateID,
				BidsToUpdate:  transformRawBids(evt.BidsToUpdate),
				AsksToUpdate:  transformRawBids(evt.AsksToUpdate),
			}
		}
		n, err = ws.Read(msg)
	}
}

func (b *binanceStream) getDepth(symbol string, limit int) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.binance.com/api/v1/depth?symbol=%s&limit=%d", symbol, limit), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rawData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	depth := &depth{}
	err = json.Unmarshal(rawData, depth)
	if err != nil {
		return err
	}
	b.LastUpdateID = depth.LastUpdateID
	return nil
}
