export interface ApiError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}

export interface User {
  id: string;
  email: string;
  username: string;
  display_name: string;
  role: string;
  status: string;
  email_verified: boolean;
  created_at: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

export interface Channel {
  id: string;
  user_id: string;
  slug: string;
  title: string;
  description: string;
  banner_url: string;
  avatar_url: string;
  category_id: string;
  category_slug: string;
  is_verified: boolean;
  is_live: boolean;
  follower_count: number;
  created_at_unix: number;
  updated_at_unix: number;
}

export interface Stream {
  id: string;
  channel_id: string;
  channel_slug: string;
  channel_title: string;
  title: string;
  description: string;
  thumbnail_url: string;
  status: string;
  ingest_protocol: string;
  latency_mode: string;
  visibility: string;
  category_id: string;
  tags: string[] | null;
  viewer_count: number;
  peak_viewers: number;
  scheduled_at_unix: number;
  started_at_unix: number;
  ended_at_unix: number;
  created_at_unix: number;
  updated_at_unix: number;
}

export interface PaginatedStreams {
  data: Stream[];
  pagination: {
    page: number;
    limit: number;
    total: number;
  };
}

export interface Playback {
  stream_id: string;
  url: string;
  format: string;
  status: string;
  expires_at_unix: number;
  latency_mode?: string;
  playback_mode?: "ll-hls" | "dual" | "whep";
  whep_url?: string;
  hls_ready?: boolean;
}

export interface IngestKey {
  stream_key: string;
  rtmp_url: string;
  srt_url: string;
  key_prefix: string;
  whip_base_url?: string;
  whip_url?: string;
}
