package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/hacel/go-chat/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	serverAddr = flag.String("server_addr", "localhost:50051", "The server address in the format of host:port")
	certFile   = flag.String("cert", "keys/ca_cert.pem", "The file containing the CA root cert file")
)

func runChat(client pb.ChatClient) {
	ctx, cancel := context.WithCancel(context.Background())
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
				log.Fatalf("Failed to receive a message: %v", err)
			}
			fmt.Printf("\b\b%s %s: %s\n> ", time.Now().Format("15:04"), in.From, in.Body)
		}
	}()
	go func() {
		stream.Send(&pb.ChatMessage{Body: "User Connected."})
		for {
			var body string
			fmt.Printf("> ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				body = scanner.Text()
			}
			msg := pb.ChatMessage{Body: body}
			if err := stream.Send(&msg); err != nil {
				log.Fatalf("Failed to send a message: %v", err)
			}
		}
	}()
	<-waitc
}

func main() {
	flag.Parse()

	creds, err := credentials.NewClientTLSFromFile(*certFile, "")
	if err != nil {
		log.Fatalf("Failed to generate credentials: %v", err)
	}

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(creds), grpc.WithBlock())
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewChatClient(conn)

	runChat(client)
}
