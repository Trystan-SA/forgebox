import { clsx } from "clsx";
import {
  Clock,
  Loader2,
  CheckCircle,
  XCircle,
  Ban,
} from "lucide-react";
import type { TaskStatus } from "../lib/types";

const config: Record<TaskStatus, { label: string; color: string; Icon: typeof Clock }> = {
  pending: { label: "Pending", color: "bg-gray-100 text-gray-700", Icon: Clock },
  running: { label: "Running", color: "bg-blue-100 text-blue-700", Icon: Loader2 },
  completed: { label: "Completed", color: "bg-green-100 text-green-700", Icon: CheckCircle },
  failed: { label: "Failed", color: "bg-red-100 text-red-700", Icon: XCircle },
  cancelled: { label: "Cancelled", color: "bg-yellow-100 text-yellow-700", Icon: Ban },
};

export function StatusBadge({ status }: { status: TaskStatus }) {
  const { label, color, Icon } = config[status] ?? config.pending;

  return (
    <span className={clsx("badge gap-1", color)}>
      <Icon
        className={clsx("h-3 w-3", status === "running" && "animate-spin")}
      />
      {label}
    </span>
  );
}
