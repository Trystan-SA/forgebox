import { useEffect, useRef } from "react";
import { clsx } from "clsx";
import { Terminal, Wrench, AlertCircle } from "lucide-react";
import type { TaskEvent } from "../lib/types";

interface TaskStreamProps {
  events: TaskEvent[];
  isRunning: boolean;
}

export function TaskStream({ events, isRunning }: TaskStreamProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [events.length]);

  if (events.length === 0 && !isRunning) {
    return null;
  }

  return (
    <div className="card overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2">
        <Terminal className="h-4 w-4 text-gray-500" />
        <span className="text-sm font-medium text-gray-700">Output</span>
        {isRunning && (
          <span className="ml-auto flex items-center gap-1 text-xs text-blue-600">
            <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-blue-600" />
            Streaming
          </span>
        )}
      </div>

      {/* Event list */}
      <div className="max-h-[500px] overflow-y-auto bg-gray-900 p-4 font-mono text-sm">
        {events.map((event, i) => (
          <EventLine key={i} event={event} />
        ))}
        {isRunning && (
          <span className="inline-block h-4 w-2 animate-pulse bg-gray-400" />
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}

function EventLine({ event }: { event: TaskEvent }) {
  switch (event.type) {
    case "text_delta":
      return <span className="text-gray-100 whitespace-pre-wrap">{event.text}</span>;

    case "tool_call":
      return (
        <div className="my-2 flex items-start gap-2 rounded border border-gray-700 bg-gray-800 p-2">
          <Wrench className="mt-0.5 h-3.5 w-3.5 shrink-0 text-forge-400" />
          <div>
            <span className="font-semibold text-forge-300">
              {event.tool_call?.name}
            </span>
            <pre className="mt-1 text-xs text-gray-400 overflow-x-auto">
              {event.tool_call?.input}
            </pre>
          </div>
        </div>
      );

    case "tool_result":
      return (
        <div
          className={clsx(
            "my-1 rounded border p-2 text-xs whitespace-pre-wrap",
            event.result?.is_error
              ? "border-red-800 bg-red-950 text-red-300"
              : "border-gray-700 bg-gray-800 text-gray-300",
          )}
        >
          {event.result?.content}
        </div>
      );

    case "error":
      return (
        <div className="my-2 flex items-center gap-2 text-red-400">
          <AlertCircle className="h-4 w-4" />
          <span>{event.error}</span>
        </div>
      );

    case "done":
      return (
        <div className="mt-3 border-t border-gray-700 pt-2 text-xs text-gray-500">
          Task completed
        </div>
      );

    default:
      return null;
  }
}
