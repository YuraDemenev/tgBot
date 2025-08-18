package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/handlers"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func startGRPCServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	// add a listener address
	lis, err := net.Listen("tcp", ":50002")
	if err != nil {
		logrus.Fatalf("ERROR STARTING THE SERVER : %v", err)
	}

	// create the grpc server
	grpcServer := grpc.NewServer()
	taskpb.RegisterTaskServiceServer(grpcServer, &handlers.TaskServer{})

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

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
	var mainWG sync.WaitGroup

	mainWG.Add(1)
	go startGRPCServer(ctx, &mainWG)
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
	mainWG.Wait()
}
