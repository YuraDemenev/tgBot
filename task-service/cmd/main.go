package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/handlers"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func startGRPCServer(ctx context.Context) {
	// add a listener address
	lis, err := net.Listen("tcp", ":50002")
	if err != nil {
		logrus.Fatalf("ERROR STARTING THE SERVER : %v", err)
	}

	// start the grpc server
	grpcServer := grpc.NewServer()
	taskpb.RegisterTaskServiceServer(grpcServer, handlers.TaskServer{})

	// start serving to the address
	go func() {
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			logrus.Fatalf("grpcServer can't serve: %v", err)
		}
	}()

	select {
	case <-ctx.Done():
		grpcServer.GracefulStop()
		return
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startGRPCServer(ctx)
	logrus.Info("Service task-service started grpc server")

	//handle graceful shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logrus.Infof("Received shutdown signal: %v", sig)
		cancel()
	}()

	logrus.Info("Service task-service started")
}
