import { apiFetch } from "./client";

export interface AdminStats {
  users: {
    total: number;
    active: number;
    suspended: number;
    banned: number;
    admins: number;
  };
  streams: {
    live_total: number;
    live_marketplace: number;
  };
}

export interface AdminUser {
  id: string;
  email: string;
  username: string;
  display_name: string;
  role: string;
  status: string;
  email_verified: boolean;
  created_at_unix: number;
}

export interface AdminChannel {
  id: string;
  slug: string;
  title: string;
  is_verified: boolean;
  is_live: boolean;
  follower_count: number;
  marketplace_seller_id?: number;
  marketplace_shop_id?: number;
  created_at_unix: number;
}

export interface AdminStream {
  id: string;
  title: string;
  channel_slug: string;
  channel_title: string;
  status: string;
  viewer_count: number;
  marketplace_seller_id?: number;
  marketplace_shop_id?: number;
  started_at_unix?: number;
}

export interface Paginated<T> {
  data: T[];
  pagination: { page: number; limit: number; total: number };
}

export interface AuditLog {
  id: number;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  details: string;
  created_at_unix: number;
}

export function getAdminStats() {
  return apiFetch<AdminStats>("/v1/admin/stats", { auth: true });
}

export function listAdminUsers(params: {
  page?: number;
  limit?: number;
  status?: string;
  role?: string;
  search?: string;
}) {
  const q = new URLSearchParams();
  if (params.page) q.set("page", String(params.page));
  if (params.limit) q.set("limit", String(params.limit));
  if (params.status) q.set("status", params.status);
  if (params.role) q.set("role", params.role);
  if (params.search) q.set("search", params.search);
  const qs = q.toString();
  return apiFetch<Paginated<AdminUser>>(`/v1/admin/users${qs ? `?${qs}` : ""}`, {
    auth: true,
  });
}

export function updateAdminUser(
  id: string,
  body: { role?: string; status?: string },
) {
  return apiFetch<AdminUser>(`/v1/admin/users/${id}`, {
    method: "PATCH",
    auth: true,
    body: JSON.stringify(body),
  });
}

export function listAdminChannels(params: {
  page?: number;
  limit?: number;
  search?: string;
  marketplace_only?: boolean;
}) {
  const q = new URLSearchParams();
  if (params.page) q.set("page", String(params.page));
  if (params.limit) q.set("limit", String(params.limit));
  if (params.search) q.set("search", params.search);
  if (params.marketplace_only) q.set("marketplace_only", "true");
  const qs = q.toString();
  return apiFetch<Paginated<AdminChannel>>(
    `/v1/admin/channels${qs ? `?${qs}` : ""}`,
    { auth: true },
  );
}

export function updateAdminChannel(slug: string, isVerified: boolean) {
  return apiFetch<AdminChannel>(`/v1/admin/channels/${slug}`, {
    method: "PATCH",
    auth: true,
    body: JSON.stringify({ is_verified: isVerified }),
  });
}

export function listAdminLiveStreams(params: {
  page?: number;
  limit?: number;
  marketplace_only?: boolean;
}) {
  const q = new URLSearchParams();
  if (params.page) q.set("page", String(params.page));
  if (params.limit) q.set("limit", String(params.limit));
  if (params.marketplace_only) q.set("marketplace_only", "true");
  const qs = q.toString();
  return apiFetch<{ data: AdminStream[]; pagination: Paginated<AdminStream>["pagination"] }>(
    `/v1/admin/streams/live${qs ? `?${qs}` : ""}`,
    { auth: true },
  );
}

export function forceEndStream(id: string) {
  return apiFetch<AdminStream>(`/v1/admin/streams/${id}/force-end`, {
    method: "POST",
    auth: true,
  });
}

export function listAuditLogs(params: { page?: number; limit?: number }) {
  const q = new URLSearchParams();
  if (params.page) q.set("page", String(params.page));
  if (params.limit) q.set("limit", String(params.limit));
  const qs = q.toString();
  return apiFetch<Paginated<AuditLog>>(
    `/v1/admin/audit-logs${qs ? `?${qs}` : ""}`,
    { auth: true },
  );
}
