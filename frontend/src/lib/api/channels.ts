import { apiFetch } from "./client";
import type { Channel, IngestKey } from "@/types";

export async function getChannel(slug: string) {
  return apiFetch<Channel>(`/v1/channels/${slug}`);
}

export async function getMyChannel() {
  return apiFetch<Channel>("/v1/channels/me", { auth: true });
}

export async function createChannel(data: {
  slug: string;
  title: string;
  description?: string;
}) {
  return apiFetch<Channel>("/v1/channels", {
    method: "POST",
    auth: true,
    body: JSON.stringify(data),
  });
}

export async function followChannel(slug: string) {
  return apiFetch(`/v1/channels/${slug}/follow`, {
    method: "POST",
    auth: true,
  });
}

export async function unfollowChannel(slug: string) {
  return apiFetch(`/v1/channels/${slug}/follow`, {
    method: "DELETE",
    auth: true,
  });
}

export async function getIngestKey(slug: string) {
  return apiFetch<IngestKey>(`/v1/channels/${slug}/ingest`, { auth: true });
}

export async function rotateIngestKey(slug: string) {
  return apiFetch<IngestKey>(`/v1/channels/${slug}/key/rotate`, {
    method: "POST",
    auth: true,
  });
}
