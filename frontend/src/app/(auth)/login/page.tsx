"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { login } from "@/lib/api/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { useState } from "react";

const schema = z.object({
  email: z.string().email("Email noto'g'ri"),
  password: z.string().min(8, "Kamida 8 belgi"),
});

type Form = z.infer<typeof schema>;

export default function LoginPage() {
  const router = useRouter();
  const [error, setError] = useState("");

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Form>({ resolver: zodResolver(schema) });

  async function onSubmit(data: Form) {
    setError("");
    try {
      await login(data);
      router.push("/");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Kirish muvaffaqiyatsiz");
    }
  }

  return (
    <Card>
      <CardHeader>
        <h1 className="text-2xl font-bold">Kirish</h1>
        <p className="text-sm text-muted">Hisobingizga kiring</p>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium">Email</label>
            <Input type="email" placeholder="you@email.com" {...register("email")} />
            {errors.email && (
              <p className="mt-1 text-xs text-red-400">{errors.email.message}</p>
            )}
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium">Parol</label>
            <Input type="password" placeholder="••••••••" {...register("password")} />
            {errors.password && (
              <p className="mt-1 text-xs text-red-400">{errors.password.message}</p>
            )}
          </div>
          {error && (
            <p className="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</p>
          )}
          <Button type="submit" className="w-full" loading={isSubmitting}>
            Kirish
          </Button>
        </form>
        <p className="mt-6 text-center text-sm text-muted">
          Hisobingiz yo&apos;qmi?{" "}
          <Link href="/register" className="text-accent hover:underline">
            Ro&apos;yxatdan o&apos;tish
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
