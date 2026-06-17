"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listAdminUsers, updateAdminUser } from "@/lib/api/admin";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

export default function AdminUsersPage() {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["admin-users", page, search, status],
    queryFn: () => listAdminUsers({ page, limit: 20, search, status }),
  });

  const mutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: { role?: string; status?: string } }) =>
      updateAdminUser(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-users"] }),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Foydalanuvchilar</h1>
        <p className="text-muted">Rol va status boshqaruvi</p>
      </div>

      <div className="flex flex-wrap gap-2">
        <Input
          placeholder="Qidirish (email, username)"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
          className="max-w-xs"
        />
        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            setPage(1);
          }}
          className="rounded-xl border border-border bg-surface-1 px-3 py-2 text-sm"
        >
          <option value="">Barcha status</option>
          <option value="active">active</option>
          <option value="suspended">suspended</option>
          <option value="banned">banned</option>
        </select>
      </div>

      <Card>
        <CardHeader>
          <h2 className="font-semibold">Ro&apos;yxat</h2>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          {isLoading ? (
            <Skeleton className="h-40 w-full" />
          ) : (
            <table className="w-full min-w-[720px] text-left text-sm">
              <thead>
                <tr className="border-b border-border text-muted">
                  <th className="pb-2 pr-4">User</th>
                  <th className="pb-2 pr-4">Rol</th>
                  <th className="pb-2 pr-4">Status</th>
                  <th className="pb-2">Amallar</th>
                </tr>
              </thead>
              <tbody>
                {data?.data.map((u) => (
                  <tr key={u.id} className="border-b border-border/60">
                    <td className="py-3 pr-4">
                      <div className="font-medium">{u.display_name || u.username}</div>
                      <div className="text-xs text-muted">
                        {u.email} · @{u.username}
                      </div>
                    </td>
                    <td className="py-3 pr-4">
                      <Badge variant="outline">{u.role}</Badge>
                    </td>
                    <td className="py-3 pr-4">
                      <Badge variant={u.status === "active" ? "default" : "live"}>
                        {u.status}
                      </Badge>
                    </td>
                    <td className="py-3">
                      <div className="flex flex-wrap gap-1">
                        {u.status !== "active" && (
                          <Button
                            size="sm"
                            variant="secondary"
                            disabled={mutation.isPending}
                            onClick={() =>
                              mutation.mutate({ id: u.id, body: { status: "active" } })
                            }
                          >
                            Faollashtirish
                          </Button>
                        )}
                        {u.status !== "suspended" && (
                          <Button
                            size="sm"
                            variant="secondary"
                            disabled={mutation.isPending}
                            onClick={() =>
                              mutation.mutate({ id: u.id, body: { status: "suspended" } })
                            }
                          >
                            To&apos;xtatish
                          </Button>
                        )}
                        {u.role !== "admin" && (
                          <Button
                            size="sm"
                            disabled={mutation.isPending}
                            onClick={() =>
                              mutation.mutate({ id: u.id, body: { role: "admin" } })
                            }
                          >
                            Admin qilish
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {data && data.pagination.total > 20 && (
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            Oldingi
          </Button>
          <Button
            variant="secondary"
            size="sm"
            disabled={page * 20 >= data.pagination.total}
            onClick={() => setPage((p) => p + 1)}
          >
            Keyingi
          </Button>
        </div>
      )}
    </div>
  );
}
