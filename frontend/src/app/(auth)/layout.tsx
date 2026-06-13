import Link from "next/link";
import { Radio } from "lucide-react";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen">
      <div className="brand-bg hidden w-1/2 flex-col justify-between p-12 text-white lg:flex">
        <Link href="/" className="flex items-center gap-2">
          <div className="icon-gold-gradient flex h-10 w-10 items-center justify-center rounded-xl">
            <Radio className="h-5 w-5 text-brand" />
          </div>
          <span className="text-xl font-bold">
            Sahiy<span className="text-gradient-gold">Stream</span>
          </span>
        </Link>
        <div>
          <h2 className="text-4xl font-bold leading-tight">
            Millionlab tomoshabinlar sizni kutmoqda
          </h2>
          <p className="mt-4 text-lg text-white/70">
            Professional streaming, past latency, ABR sifat.
          </p>
        </div>
        <p className="text-sm text-white/40">© 2026 Sahiy Stream</p>
      </div>
      <div className="flex flex-1 items-center justify-center p-6">
        <div className="w-full max-w-md">{children}</div>
      </div>
    </div>
  );
}
