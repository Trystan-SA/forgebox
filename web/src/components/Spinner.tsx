import { Loader2 } from "lucide-react";
import { clsx } from "clsx";

export function Spinner({ className }: { className?: string }) {
  return (
    <Loader2
      className={clsx("animate-spin text-forge-600", className ?? "h-5 w-5")}
    />
  );
}

export function PageSpinner() {
  return (
    <div className="flex items-center justify-center py-20">
      <Spinner className="h-8 w-8" />
    </div>
  );
}
