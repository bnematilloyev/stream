package grpc

import (
	"context"

	"github.com/google/uuid"
	chatv1 "github.com/sahiy/sahiy-stream/proto/gen/chat/v1"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/usecase"
)

type ChatServer struct {
	chatv1.UnimplementedChatServiceServer
	uc *usecase.ChatUseCase
}

func NewChatServer(uc *usecase.ChatUseCase) *ChatServer {
	return &ChatServer{uc: uc}
}

func (s *ChatServer) GetHistory(ctx context.Context, req *chatv1.GetHistoryRequest) (*chatv1.GetHistoryResponse, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	list, hasMore, err := s.uc.GetHistory(ctx, streamID, req.GetBeforeId(), int(req.GetLimit()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*chatv1.ChatMessage, 0, len(list))
	for i := len(list) - 1; i >= 0; i-- {
		out = append(out, toProto(list[i]))
	}
	return &chatv1.GetHistoryResponse{Messages: out, HasMore: hasMore}, nil
}

func (s *ChatServer) DeleteMessage(ctx context.Context, req *chatv1.DeleteMessageRequest) (*chatv1.DeleteMessageResponse, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	if err := s.uc.DeleteMessage(ctx, streamID, req.GetMessageId(), req.GetActorUserId(), req.GetActorRole()); err != nil {
		return nil, toGRPCError(err)
	}
	return &chatv1.DeleteMessageResponse{Success: true}, nil
}

func toProto(m domain.Message) *chatv1.ChatMessage {
	msg := &chatv1.ChatMessage{
		Id: m.ID, StreamId: m.StreamID.String(), Username: m.Username,
		DisplayName: m.DisplayName, Content: m.Content, Type: m.Type,
		CreatedAtUnix: m.CreatedAt.Unix(),
	}
	if m.UserID != nil {
		msg.UserId = m.UserID.String()
	}
	return msg
}
