import { formatDistanceToNow } from "date-fns";
import { Cpu, DollarSign } from "lucide-react";
import type { Task } from "../lib/types";
import { StatusBadge } from "./StatusBadge";

interface TaskCardProps {
  task: Task;
  onClick?: () => void;
}

export function TaskCard({ task, onClick }: TaskCardProps) {
  return (
    <button
      onClick={onClick}
      className="card w-full p-4 text-left transition-shadow hover:shadow-md"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-gray-900">
            {task.prompt}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-gray-500">
            <span className="flex items-center gap-1">
              <Cpu className="h-3 w-3" />
              {task.provider}
              {task.model && ` / ${task.model}`}
            </span>
            {task.cost > 0 && (
              <span className="flex items-center gap-1">
                <DollarSign className="h-3 w-3" />
                ${task.cost.toFixed(4)}
              </span>
            )}
            <span>
              {formatDistanceToNow(new Date(task.created_at), {
                addSuffix: true,
              })}
            </span>
          </div>
        </div>
        <StatusBadge status={task.status} />
      </div>
    </button>
  );
}
