package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/pagination"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/usecase"
)

type StreamServer struct {
	streamv1.UnimplementedStreamServiceServer
	uc       *usecase.StreamUseCase
	playback *usecase.PlaybackUseCase
	viewers  *usecase.ViewerUseCase
}

func NewStreamServer(uc *usecase.StreamUseCase, playback *usecase.PlaybackUseCase, viewers *usecase.ViewerUseCase) *StreamServer {
	return &StreamServer{uc: uc, playback: playback, viewers: viewers}
}

func (s *StreamServer) CreateStream(ctx context.Context, req *streamv1.CreateStreamRequest) (*streamv1.Stream, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	var scheduled *time.Time
	if req.GetScheduledAtUnix() > 0 {
		t := time.Unix(req.GetScheduledAtUnix(), 0)
		scheduled = &t
	}
	st, err := s.uc.Create(ctx, userID, req.GetChannelSlug(), req.GetTitle(), req.GetDescription(),
		req.GetIngestProtocol(), req.GetLatencyMode(), req.GetVisibility(), req.GetCategoryId(), req.GetTags(), scheduled)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) GetStream(ctx context.Context, req *streamv1.GetStreamRequest) (*streamv1.Stream, error) {
	id, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	st, err := s.uc.Get(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) UpdateStream(ctx context.Context, req *streamv1.UpdateStreamRequest) (*streamv1.Stream, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	var catID *uuid.UUID
	if req.CategoryId != nil && *req.CategoryId != "" {
		id, err := uuid.Parse(*req.CategoryId)
		if err != nil {
			return nil, toGRPCError(err)
		}
		catID = &id
	}
	st, err := s.uc.Update(ctx, userID, streamID, req.Title, req.Description, req.Visibility, catID, req.GetTags())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) DeleteStream(ctx context.Context, req *streamv1.DeleteStreamRequest) (*streamv1.DeleteStreamResponse, error) {
	userID, _ := uuid.Parse(req.GetUserId())
	streamID, _ := uuid.Parse(req.GetStreamId())
	if err := s.uc.Delete(ctx, userID, streamID); err != nil {
		return nil, toGRPCError(err)
	}
	return &streamv1.DeleteStreamResponse{Success: true}, nil
}

func (s *StreamServer) ListLiveStreams(ctx context.Context, req *streamv1.ListLiveStreamsRequest) (*streamv1.ListStreamsResponse, error) {
	list, meta, err := s.uc.ListLive(ctx, int(req.GetPage()), int(req.GetLimit()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toList(list, meta), nil
}

func (s *StreamServer) ListChannelStreams(ctx context.Context, req *streamv1.ListChannelStreamsRequest) (*streamv1.ListStreamsResponse, error) {
	list, meta, err := s.uc.ListByChannel(ctx, req.GetChannelSlug(), req.GetStatus(), int(req.GetPage()), int(req.GetLimit()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toList(list, meta), nil
}

func (s *StreamServer) StartStream(ctx context.Context, req *streamv1.StartStreamRequest) (*streamv1.Stream, error) {
	userID, _ := uuid.Parse(req.GetUserId())
	streamID, _ := uuid.Parse(req.GetStreamId())
	st, err := s.uc.Start(ctx, userID, streamID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) EndStream(ctx context.Context, req *streamv1.EndStreamRequest) (*streamv1.Stream, error) {
	userID, _ := uuid.Parse(req.GetUserId())
	streamID, _ := uuid.Parse(req.GetStreamId())
	st, err := s.uc.End(ctx, userID, streamID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) ValidateStreamKey(ctx context.Context, req *streamv1.ValidateStreamKeyRequest) (*streamv1.ValidateStreamKeyResponse, error) {
	res, err := s.uc.ValidateStreamKey(ctx, req.GetStreamKey())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &streamv1.ValidateStreamKeyResponse{
		Valid:       res.Valid,
		ChannelId:   res.ChannelID.String(),
		ChannelSlug: res.ChannelSlug,
		StreamId:    res.StreamID,
	}, nil
}

func (s *StreamServer) GetPlayback(ctx context.Context, req *streamv1.GetPlaybackRequest) (*streamv1.PlaybackResponse, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	res, err := s.playback.GetPlayback(ctx, streamID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &streamv1.PlaybackResponse{
		StreamId: res.StreamID.String(), Url: res.URL, Format: res.Format,
		Status: res.Status, ExpiresAtUnix: res.ExpiresAt.Unix(),
	}, nil
}

func (s *StreamServer) StartIngest(ctx context.Context, req *streamv1.StartIngestRequest) (*streamv1.Stream, error) {
	channelID, err := uuid.Parse(req.GetChannelId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	st, err := s.uc.StartIngest(ctx, channelID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) EndIngest(ctx context.Context, req *streamv1.EndIngestRequest) (*streamv1.Stream, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	st, err := s.uc.EndIngest(ctx, streamID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) GetScheduledForChannel(ctx context.Context, req *streamv1.GetScheduledForChannelRequest) (*streamv1.Stream, error) {
	channelID, err := uuid.Parse(req.GetChannelId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	st, err := s.uc.GetScheduledForChannel(ctx, channelID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProto(st), nil
}

func (s *StreamServer) RecordViewerHeartbeat(ctx context.Context, req *streamv1.RecordViewerHeartbeatRequest) (*streamv1.ViewerStatsResponse, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	stats, err := s.viewers.Heartbeat(ctx, streamID, req.GetSessionId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &streamv1.ViewerStatsResponse{
		StreamId: streamID.String(), Concurrent: stats.Concurrent, Unique: stats.Unique,
	}, nil
}

func (s *StreamServer) GetViewerStats(ctx context.Context, req *streamv1.GetViewerStatsRequest) (*streamv1.ViewerStatsResponse, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	stats, err := s.viewers.Stats(ctx, streamID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &streamv1.ViewerStatsResponse{
		StreamId: streamID.String(), Concurrent: stats.Concurrent, Unique: stats.Unique,
	}, nil
}

func toList(list []domain.Stream, meta pagination.Result) *streamv1.ListStreamsResponse {
	out := &streamv1.ListStreamsResponse{
		Total: int32(meta.Total), Page: int32(meta.Page), Limit: int32(meta.Limit),
	}
	for _, st := range list {
		out.Streams = append(out.Streams, toProto(&st))
	}
	return out
}

func toProto(st *domain.Stream) *streamv1.Stream {
	out := &streamv1.Stream{
		Id: st.ID.String(), ChannelId: st.ChannelID.String(), ChannelSlug: st.ChannelSlug,
		ChannelTitle: st.ChannelTitle, Title: st.Title, Status: st.Status,
		IngestProtocol: st.IngestProtocol, LatencyMode: st.LatencyMode, Visibility: st.Visibility,
		Tags: st.Tags, ViewerCount: int32(st.ViewerCount), PeakViewers: int32(st.PeakViewers),
		CreatedAtUnix: st.CreatedAt.Unix(), UpdatedAtUnix: st.UpdatedAt.Unix(),
	}
	if st.Description != nil {
		out.Description = *st.Description
	}
	if st.ThumbnailURL != nil {
		out.ThumbnailUrl = *st.ThumbnailURL
	}
	if st.CategoryID != nil {
		out.CategoryId = st.CategoryID.String()
	}
	if st.ScheduledAt != nil {
		out.ScheduledAtUnix = st.ScheduledAt.Unix()
	}
	if st.StartedAt != nil {
		out.StartedAtUnix = st.StartedAt.Unix()
	}
	if st.EndedAt != nil {
		out.EndedAtUnix = st.EndedAt.Unix()
	}
	return out
}
