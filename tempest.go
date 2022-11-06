package main

import (
	"context"
	"fmt"
	"hacknc2022/protobuf"
	"io"
	"log"
	"net"

	"github.com/go-numb/go-ftx/realtime"
	"google.golang.org/grpc"
)

type TradeServer struct {
	protobuf.UnimplementedTradeServiceServer
}

// Sends Tickers to all active bots & awaits their signals
// Currently only works on 1 bot at a time & trades with 1 share/crypto unit
func (t *TradeServer) StreamTicker(stream protobuf.TradeService_StreamTickerServer) error {
	// Initiator
	if err := stream.Send(&protobuf.Ticker{Value: 0.0, Bs: false}); err != nil {
		return err
	}

	waitc := make(chan struct{})

	var startingCapital float64 = 100000.0
	var currentCapital float64 = startingCapital
	var openPosition bool = false
	var normalizedCurrentPrice float64 = 0

	ch := make(chan realtime.Response)
	go realtime.Connect(context.Background(), ch, []string{"ticker"}, []string{"BTC/USD"}, nil)

	// Sender
	go func() {
		for {
			v := <-ch

			if v.Ticker.Last != normalizedCurrentPrice {
				normalizedCurrentPrice = v.Ticker.Last

				if err := stream.Send(&protobuf.Ticker{Value: float32(normalizedCurrentPrice), Bs: openPosition}); err != nil {
					panic(err) // will fail if client disconnects, ok for now
				}
			}
		}
	}()

	// Reciever
	go func() {
		for {
			value, err := stream.Recv()

			if err == io.EOF {
				log.Println("EOF")
				return
			}
			if err != nil {
				panic(err) // will fail if client disconnects, ok for now
			}

			signal := value.GetValue()

			// 0 == no signal, 1 == buy, 2 == sell open positions
			switch signal {
			case 0:
				break
			case 1:
				if !openPosition { // Buy if we have no open position
					fmt.Println("Bought")
					openPosition = true
					currentCapital = currentCapital - normalizedCurrentPrice
				}
			case 2:
				if openPosition { // Sell if we have open position
					fmt.Println("Sold")
					openPosition = false
					currentCapital = currentCapital + normalizedCurrentPrice
				}
			}
			fmt.Printf("Current Capital %f\n", currentCapital)
		}
	}()

	<-waitc
	return nil
}

func main() {
	lis, err := net.Listen("tcp", "localhost:9111")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	protobuf.RegisterTradeServiceServer(grpcServer, &TradeServer{})
	log.Printf("Initializing gRPC server on port :9111")
	grpcServer.Serve(lis)
}
