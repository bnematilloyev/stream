"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { listAuditLogs } from "@/lib/api/admin";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export default function AdminAuditPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit", page],
    queryFn: () => listAuditLogs({ page, limit: 30 }),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Audit log</h1>
        <p className="text-muted">Admin amallar tarixi</p>
      </div>

      <Card>
        <CardHeader>
          <h2 className="font-semibold">So&apos;nggi yozuvlar</h2>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          {isLoading ? (
            <Skeleton className="h-40 w-full" />
          ) : (
            <table className="w-full min-w-[640px] text-left text-sm">
              <thead>
                <tr className="border-b border-border text-muted">
                  <th className="pb-2 pr-4">Vaqt</th>
                  <th className="pb-2 pr-4">Action</th>
                  <th className="pb-2 pr-4">Resource</th>
                  <th className="pb-2">Details</th>
                </tr>
              </thead>
              <tbody>
                {data?.data.map((log) => (
                  <tr key={log.id} className="border-b border-border/60">
                    <td className="py-2 pr-4 text-xs text-muted">
                      {new Date(log.created_at_unix * 1000).toLocaleString()}
                    </td>
                    <td className="py-2 pr-4 font-mono text-xs">{log.action}</td>
                    <td className="py-2 pr-4 text-xs">
                      {log.resource_type}
                      {log.resource_id ? ` / ${log.resource_id.slice(0, 8)}…` : ""}
                    </td>
                    <td className="py-2 max-w-xs truncate text-xs text-muted">
                      {log.details}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {data && data.pagination.total > 30 && (
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
            disabled={page * 30 >= data.pagination.total}
            onClick={() => setPage((p) => p + 1)}
          >
            Keyingi
          </Button>
        </div>
      )}
    </div>
  );
}
