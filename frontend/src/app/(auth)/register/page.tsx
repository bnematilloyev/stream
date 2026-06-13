"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { register as registerApi } from "@/lib/api/auth";
import { useAuthStore } from "@/stores/authStore";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { useState } from "react";

const schema = z.object({
  email: z.string().email("Email noto'g'ri"),
  username: z
    .string()
    .min(3, "Kamida 3 belgi")
    .max(30)
    .regex(/^[a-zA-Z0-9_]+$/, "Faqat harf, raqam va _"),
  display_name: z.string().min(2, "Kamida 2 belgi").max(50),
  password: z.string().min(8, "Kamida 8 belgi"),
});

type Form = z.infer<typeof schema>;

export default function RegisterPage() {
  const router = useRouter();
  const setAuth = useAuthStore((s) => s.setAuth);
  const [error, setError] = useState("");

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Form>({ resolver: zodResolver(schema) });

  async function onSubmit(data: Form) {
    setError("");
    try {
      const res = await registerApi(data);
      setAuth(res.user, res.access_token);
      router.push("/studio");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ro'yxatdan o'tish muvaffaqiyatsiz");
    }
  }

  return (
    <Card>
      <CardHeader>
        <h1 className="text-2xl font-bold">Ro&apos;yxatdan o&apos;tish</h1>
        <p className="text-sm text-muted">Yangi hisob yarating</p>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium">Email</label>
            <Input type="email" {...register("email")} />
            {errors.email && (
              <p className="mt-1 text-xs text-red-400">{errors.email.message}</p>
            )}
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium">Username</label>
            <Input placeholder="creator1" {...register("username")} />
            {errors.username && (
              <p className="mt-1 text-xs text-red-400">{errors.username.message}</p>
            )}
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium">Ko&apos;rinadigan ism</label>
            <Input placeholder="Creator" {...register("display_name")} />
            {errors.display_name && (
              <p className="mt-1 text-xs text-red-400">{errors.display_name.message}</p>
            )}
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium">Parol</label>
            <Input type="password" {...register("password")} />
            {errors.password && (
              <p className="mt-1 text-xs text-red-400">{errors.password.message}</p>
            )}
          </div>
          {error && (
            <p className="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</p>
          )}
          <Button type="submit" className="w-full" loading={isSubmitting}>
            Yaratish
          </Button>
        </form>
        <p className="mt-6 text-center text-sm text-muted">
          Hisobingiz bormi?{" "}
          <Link href="/login" className="text-accent hover:underline">
            Kirish
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
