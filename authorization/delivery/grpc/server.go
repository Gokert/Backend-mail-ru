package delivery_auth_grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/repository/profile"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/repository/session"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"google.golang.org/grpc"

	pb "github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/proto"
)

type authGrpc struct {
	grpcServ *grpc.Server
	lg       *slog.Logger
}

type server struct {
	pb.UnimplementedAuthorizationServer
	userRepo    *profile.RepoPostgre
	sessionRepo *session.SessionRepo
	lg          *slog.Logger
}

func NewServer(l *slog.Logger) (*authGrpc, error) {
	config, err := configs.ReadConfig()
	if err != nil {
		l.Error("read config error", "err", err.Error())
		return nil, fmt.Errorf("listen and serve grpc error: %w", err)
	}

	configSession, err := configs.ReadSessionRedisConfig()
	if err != nil {
		l.Error("read config error", "err", err.Error())
		return nil, fmt.Errorf("listen and serve grpc error: %w", err)
	}

	session, err := session.GetSessionRepo(*configSession, l)

	if err != nil {
		l.Error("Session repository is not responding")
		return nil, fmt.Errorf("listen and serve grpc error: %w", err)
	}

	users, err := profile.GetUserRepo(config, l)
	if err != nil {
		l.Error("cant create repo")
		return nil, fmt.Errorf("listen and serve grpc error: %w", err)
	}

	s := grpc.NewServer()
	pb.RegisterAuthorizationServer(s, &server{
		lg:          l,
		sessionRepo: session,
		userRepo:    users,
	})

	return &authGrpc{grpcServ: s, lg: l}, nil
}

func (s *server) GetId(ctx context.Context, req *pb.FindIdRequest) (*pb.FindIdResponse, error) {
	login, err := s.sessionRepo.GetUserLogin(ctx, req.Sid, s.lg)
	if err != nil {
		return nil, err
	}

	id, err := s.userRepo.GetUserProfileId(login)
	if err != nil {
		s.lg.Error("failed get user profile id: %v", err)
		return nil, err
	}
	return &pb.FindIdResponse{
		Value: id,
	}, nil
}

func (s *server) GetIdsAndPaths(ctx context.Context, req *pb.NamesAndPathsListRequest) (*pb.NamesAndPathsResponse, error) {
	names, paths, err := s.userRepo.GetNamesAndPaths(req.Ids)
	if err != nil {
		s.lg.Error("failed get users ids and photo: %v", err)
		return nil, err
	}
	return &pb.NamesAndPathsResponse{
		Names: names,
		Paths: paths,
	}, nil
}

func (s *server) GetAuthorizationStatus(ctx context.Context, req *pb.AuthorizationCheckRequest) (*pb.AuthorizationCheckResponse, error) {
	status, err := s.sessionRepo.CheckActiveSession(ctx, req.Sid, s.lg)
	if err != nil {
		s.lg.Error("failed to check auth status: %v", err)
		return nil, err
	}
	return &pb.AuthorizationCheckResponse{
		Status: status,
	}, nil
}

func (s *server) GetRole(ctx context.Context, req *pb.RoleRequest) (*pb.RoleResponse, error) {
    role, err := s.userRepo.GetUserRole(req.Login)
	if err != nil {
		s.lg.Error("failed to get user role: %v", err)
		return nil, err
	}

	return &pb.RoleResponse{
		Role: role,
	},nil
}

func (s *authGrpc) ListenAndServeGrpc() error {
	grpcConfig, err := configs.ReadGrpcConfig()
	if err != nil {
		s.lg.Error("failed to parse grpc config file: %v", err)
		return fmt.Errorf("listen and serve grpc error: %w", err)
	}

	lis, err := net.Listen(grpcConfig.ConnectionType, ":"+grpcConfig.Port)
	if err != nil {
		s.lg.Error("failed to listen: %v", err)
		return fmt.Errorf("listen and serve grpc error: %w", err)
	}

	if err := s.grpcServ.Serve(lis); err != nil {
		s.lg.Error("failed to serve: %v", err)
		return fmt.Errorf("listen and serve grpc error: %w", err)
	}

	return nil
}
