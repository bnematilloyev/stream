"use client";

import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { createChannel } from "@/lib/api/channels";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

export function CreateChannelForm() {
  const [slug, setSlug] = useState("");
  const [title, setTitle] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const qc = useQueryClient();

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      await createChannel({ slug, title });
      await qc.invalidateQueries({ queryKey: ["my-channel"] });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Xatolik");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={submit} className="max-w-md space-y-4">
      <div>
        <label className="mb-1.5 block text-sm font-medium">Kanal nomi</label>
        <Input value={title} onChange={(e) => setTitle(e.target.value)} required />
      </div>
      <div>
        <label className="mb-1.5 block text-sm font-medium">Slug (URL)</label>
        <Input
          value={slug}
          onChange={(e) => setSlug(e.target.value.toLowerCase())}
          placeholder="mychannel"
          required
        />
      </div>
      {error && <p className="text-sm text-red-400">{error}</p>}
      <Button type="submit" loading={loading}>
        Kanal yaratish
      </Button>
    </form>
  );
}
