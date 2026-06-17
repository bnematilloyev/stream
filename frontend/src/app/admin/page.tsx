"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { getAdminStats } from "@/lib/api/admin";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Users, Radio, ShoppingBag, Shield } from "lucide-react";

export default function AdminDashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["admin-stats"],
    queryFn: getAdminStats,
  });

  if (isLoading) {
    return <Skeleton className="h-64 w-full rounded-2xl" />;
  }

  const stats = data!;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Platform boshqaruvi</h1>
        <p className="text-muted">Sahiy Stream admin paneli</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          icon={Users}
          label="Foydalanuvchilar"
          value={stats.users.total}
          hint={`${stats.users.active} faol`}
        />
        <StatCard
          icon={Shield}
          label="Adminlar"
          value={stats.users.admins}
          hint={`${stats.users.suspended} to'xtatilgan`}
        />
        <StatCard
          icon={Radio}
          label="Jonli efirlar"
          value={stats.streams.live_total}
          hint="Hozir live"
        />
        <StatCard
          icon={ShoppingBag}
          label="Marketplace live"
          value={stats.streams.live_marketplace}
          hint="Seller efirlari"
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <h2 className="font-semibold">Tezkor amallar</h2>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            <Link href="/admin/users">
              <Button variant="secondary" size="sm">
                Foydalanuvchilar
              </Button>
            </Link>
            <Link href="/admin/streams">
              <Button variant="secondary" size="sm">
                Jonli monitoring
              </Button>
            </Link>
            <Link href="/admin/channels?marketplace_only=true">
              <Button variant="secondary" size="sm">
                Marketplace kanallar
              </Button>
            </Link>
            <Link href="/admin/audit">
              <Button variant="secondary" size="sm">
                Audit log
              </Button>
            </Link>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <h2 className="font-semibold">Foydalanuvchi holati</h2>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <Row label="Faol" value={stats.users.active} badge="default" />
            <Row label="To'xtatilgan" value={stats.users.suspended} badge="outline" />
            <Row label="Bloklangan" value={stats.users.banned} badge="live" />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function StatCard({
  icon: Icon,
  label,
  value,
  hint,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: number;
  hint: string;
}) {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-sm text-muted">{label}</p>
            <p className="mt-1 text-3xl font-bold">{value}</p>
            <p className="mt-1 text-xs text-muted">{hint}</p>
          </div>
          <div className="rounded-xl bg-brand-light p-2 text-brand">
            <Icon className="h-5 w-5" />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function Row({
  label,
  value,
  badge,
}: {
  label: string;
  value: number;
  badge: "default" | "outline" | "live";
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-muted">{label}</span>
      <Badge variant={badge}>{value}</Badge>
    </div>
  );
}
