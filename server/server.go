package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"

	pb "github.com/hacel/go-chat/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

var (
	port = flag.String("port", "localhost:10000", "The server port")
)

type server struct {
	pb.UnimplementedChatServer
	chatMessages []*pb.ChatMessage
	clients      map[net.Addr]pb.Chat_ChatServer
}

func (s *server) Greeting(ctx context.Context, msg *pb.ChatMessage) (*pb.ChatMessage, error) {
	return &pb.ChatMessage{From: "Server", Body: "Greetings."}, nil
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
			client.Send(msg)
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterChatServer(s, &server{clients: make(map[net.Addr]pb.Chat_ChatServer)})
	log.Printf("Listening...\n")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
