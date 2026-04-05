import { useEffect, useRef, useState } from "react";
import { Play, Square } from "lucide-react";
import { createTask, streamTask, cancelTask, listProviders } from "../lib/api";
import type { Provider, TaskEvent } from "../lib/types";
import { TaskStream } from "../components/TaskStream";

export function TaskRunner() {
  const [prompt, setPrompt] = useState("");
  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [memoryMb, setMemoryMb] = useState(512);
  const [networkAccess, setNetworkAccess] = useState(false);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [events, setEvents] = useState<TaskEvent[]>([]);
  const [isRunning, setIsRunning] = useState(false);
  const [taskId, setTaskId] = useState<string | null>(null);
  const [cost, setCost] = useState<number | null>(null);
  const [duration, setDuration] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const stopRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    listProviders()
      .then(setProviders)
      .catch(() => {});
  }, []);

  async function handleRun() {
    if (!prompt.trim() || isRunning) return;
    setEvents([]);
    setCost(null);
    setDuration(null);
    setError(null);
    setIsRunning(true);

    try {
      const start = Date.now();
      const res = await createTask({
        prompt: prompt.trim(),
        provider: provider || undefined,
        model: model || undefined,
        memory_mb: memoryMb,
        network_access: networkAccess,
      });
      setTaskId(res.task_id);

      stopRef.current = streamTask(
        res.task_id,
        (event) => {
          setEvents((prev) => [...prev, event]);
          if (event.type === "done") {
            setIsRunning(false);
            setDuration(`${((Date.now() - start) / 1000).toFixed(1)}s`);
          }
        },
        (err) => {
          setError(err.message);
          setIsRunning(false);
        },
      );
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to create task");
      setIsRunning(false);
    }
  }

  async function handleCancel() {
    if (!taskId) return;
    stopRef.current?.();
    try {
      await cancelTask(taskId);
    } catch { /* ignore */ }
    setIsRunning(false);
  }

  return (
    <>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Run Task</h1>
        <p className="mt-1 text-sm text-gray-500">
          Describe what you want to do and ForgeBox will execute it in a secure VM.
        </p>
      </div>

      {/* Prompt */}
      <textarea
        className="input mb-4"
        rows={4}
        placeholder="Describe what you want to do in plain English..."
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        disabled={isRunning}
      />

      {/* Settings row */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <select className="input w-44" value={provider} onChange={(e) => setProvider(e.target.value)} disabled={isRunning}>
          <option value="">Provider (auto)</option>
          {providers.map((p) => <option key={p.name} value={p.name}>{p.name}</option>)}
        </select>

        <input className="input w-44" placeholder="Model (default)" value={model} onChange={(e) => setModel(e.target.value)} disabled={isRunning} />

        <select className="input w-36" value={memoryMb} onChange={(e) => setMemoryMb(Number(e.target.value))} disabled={isRunning}>
          {[256, 512, 1024, 2048].map((mb) => <option key={mb} value={mb}>{mb} MB</option>)}
        </select>

        <label className="flex items-center gap-2 text-sm text-gray-600">
          <input type="checkbox" checked={networkAccess} onChange={(e) => setNetworkAccess(e.target.checked)} disabled={isRunning} className="h-4 w-4 rounded border-gray-300 text-forge-600 focus:ring-forge-500" />
          Network Access
        </label>
      </div>

      {/* Action buttons */}
      <div className="mb-6 flex items-center gap-3">
        {isRunning ? (
          <button className="btn-danger" onClick={handleCancel}>
            <Square className="mr-2 h-4 w-4" /> Cancel
          </button>
        ) : (
          <button className="btn-primary" onClick={handleRun} disabled={!prompt.trim()}>
            <Play className="mr-2 h-4 w-4" /> Run
          </button>
        )}
      </div>

      {error && <p className="mb-4 text-sm text-red-600">Error: {error}</p>}

      {/* Output */}
      <TaskStream events={events} isRunning={isRunning} />

      {/* Result summary */}
      {cost !== null && duration && (
        <div className="mt-4 flex gap-6 text-sm text-gray-500">
          <span>Cost: <strong className="text-gray-700">${cost.toFixed(4)}</strong></span>
          <span>Duration: <strong className="text-gray-700">{duration}</strong></span>
        </div>
      )}
    </>
  );
}
