package main

import (
	"context"
	"flag"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"

	pb "github.com/hacel/go-chat/chat"
)

var (
	serverAddr = flag.String("server_addr", "localhost:10000", "The server address in the format of host:port")
)

// runRouteChat receives a sequence of route notes, while sending notes for various locations.
func runChat(client pb.ChatClient) {
	msg := []string{"msg1", "msg2"}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Chat(ctx)
	if err != nil {
		log.Fatalf("%v.Chat(_) = _, %v", client, err)
	}

	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a note: %v", err)
			}
			log.Printf("Got message %s", in.Body)
		}
	}()
	for _, message := range msg {
		if err := stream.Send(message); err != nil {
			log.Fatalf("Failed to send a note: %v", err)
		}
	}
	stream.CloseSend()
	<-waitc

}

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewChatClient(conn)

	runChat(client)
}
