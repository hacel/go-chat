package main

import (
	"flag"
	"io"
	"log"
	"net"

	pb "github.com/hacel/go-chat/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

var (
	port     = flag.String("port", "localhost:50051", "The server port")
	certFile = flag.String("cert", "keys/server_cert.pem", "The TLS cert file")
	keyFile  = flag.String("key", "keys/server_key.pem", "The TLS key file")
)

type server struct {
	pb.UnimplementedChatServer
	chatMessages []*pb.ChatMessage
	clients      map[net.Addr]pb.Chat_ChatServer
}

func (s *server) Chat(stream pb.Chat_ChatServer) error {
	client, _ := peer.FromContext(stream.Context())
	clientAddr := client.Addr
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			delete(s.clients, clientAddr)
			s.Broadcast(stream, &pb.ChatMessage{From: clientAddr.String(), Body: "User Disconnected."})
			return err
		}
		log.Printf("%v: %s", clientAddr, in.Body)

		key := clientAddr
		s.clients[key] = stream
		in.From = clientAddr.String()
		s.Broadcast(stream, in)
	}
}

func (s *server) Broadcast(stream pb.Chat_ChatServer, msg *pb.ChatMessage) {
	for _, client := range s.clients {
		if client != stream {
			if err := client.Send(msg); err != nil {
				log.Fatalf("Failed to send a message to %v: %v", client, err)
			}
		}
	}
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
	if err != nil {
		log.Fatalf("Failed to generate credentials.")
	}

	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterChatServer(s, &server{clients: make(map[net.Addr]pb.Chat_ChatServer)})
	log.Printf("Listening...\n")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
