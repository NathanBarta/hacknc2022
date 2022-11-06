package main

import (
	"context"
	"hacknc2022/protobuf"
	"io"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
)

func runStreamTicker(client protobuf.TradeServiceClient) error {
	log.Printf("Streaming signals")
	ctx := context.Background()
	stream, err := client.StreamTicker(ctx)
	if err != nil {
		log.Fatalf("%v.Execute(ctx) = %v, %v: ", client, stream, err)
	}

	for {
		ticker, err := stream.Recv()
		if err == io.EOF {
			log.Println("EOF")
			return nil
		}
		if err != nil {
			log.Printf("Err: %v", err)
		}

		log.Printf("Client recieved ticker: %f", ticker.GetValue())

		// BUY / SELL Logic Here

		rand.Seed(time.Now().UnixNano())
		n := 0 + rand.Intn(2-0+1)

		if err := stream.Send(&protobuf.Signal{Value: int32(n)}); err != nil {
			return err
		}
	}
}

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial("localhost:9111", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := protobuf.NewTradeServiceClient(conn)

	runStreamTicker(client)
}
