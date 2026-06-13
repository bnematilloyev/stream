"use client";

import { useState } from "react";
import { useAuthStore } from "@/stores/authStore";
import { followChannel, unfollowChannel } from "@/lib/api/channels";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";

export function FollowButton({
  slug,
  initialFollowing = false,
}: {
  slug: string;
  initialFollowing?: boolean;
}) {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const [following, setFollowing] = useState(initialFollowing);
  const [loading, setLoading] = useState(false);

  async function toggle() {
    if (!user) {
      router.push("/login");
      return;
    }
    setLoading(true);
    try {
      if (following) {
        await unfollowChannel(slug);
        setFollowing(false);
      } else {
        await followChannel(slug);
        setFollowing(true);
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <Button
      variant={following ? "secondary" : "default"}
      size="sm"
      loading={loading}
      onClick={toggle}
    >
      {following ? "Obuna bo'lingan" : "Obuna bo'lish"}
    </Button>
  );
}
