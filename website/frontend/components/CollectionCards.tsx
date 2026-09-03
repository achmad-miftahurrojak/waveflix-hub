import Link from "next/link";
import { cn } from "@/lib/utils";

export interface CollectionCardItem {
  id: string;
  name: string;
  logoSrc?: string;
  backdropSrc?: string;
  href: string;
}

export function CollectionCards({
  items,
  className,
}: {
  items: CollectionCardItem[];
  className?: string;
}) {
  return (
    <div className={cn("grid grid-cols-1 md:grid-cols-2 gap-6", className)}>
      {items.map((item) => (
        <Link
          key={item.id}
          href={item.href}
          className="group relative block aspect-[21/9] md:aspect-[16/7] w-full overflow-hidden rounded-xl bg-background border border-white/10 transition-transform duration-300 hover:scale-[1.02] hover:border-white/30 hover:shadow-2xl"
        >
          {/* Right side Backdrop */}
          {item.backdropSrc && (
            <div className="absolute inset-y-0 right-0 w-[70%]">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={item.backdropSrc}
                alt={item.name}
                className="h-full w-full object-cover transition-transform duration-700 group-hover:scale-105"
                loading="lazy"
              />
            </div>
          )}

          {/* Gradient Overlay */}
          <div className="absolute inset-0 bg-gradient-to-r from-background via-background/90 to-transparent pointer-events-none" />

          {/* Left side Logo / Title */}
          <div className="absolute inset-y-0 left-0 flex w-[60%] flex-col justify-center p-6 md:p-8">
            {item.logoSrc ? (
              /* eslint-disable-next-line @next/next/no-img-element */
              <img
                src={item.logoSrc}
                alt={item.name}
                className="max-h-[80px] md:max-h-[100px] w-auto max-w-full object-contain object-left drop-shadow-xl filter"
              />
            ) : (
              <h3 className="text-xl md:text-3xl font-black uppercase tracking-tight text-white drop-shadow-lg" style={{ fontFamily: "var(--font-heading)" }}>
                {item.name}
              </h3>
            )}
          </div>
        </Link>
      ))}
    </div>
  );
}
