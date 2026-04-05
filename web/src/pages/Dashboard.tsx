import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  Activity,
  CheckCircle,
  XCircle,
  Clock,
  ArrowRight,
} from "lucide-react";
import { listTasks } from "../lib/api";
import type { Task } from "../lib/types";
import { StatusBadge } from "../components/StatusBadge";

export function Dashboard() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listTasks()
      .then(setTasks)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <p className="text-gray-500">Loading...</p>;
  if (error) return <p className="text-red-600">Error: {error}</p>;

  const running = tasks.filter((t) => t.status === "running").length;
  const completed = tasks.filter((t) => t.status === "completed").length;
  const failed = tasks.filter((t) => t.status === "failed").length;
  const recent = tasks.slice(0, 5);

  const stats = [
    { label: "Total Tasks", value: tasks.length, icon: Activity, color: "text-forge-600" },
    { label: "Running", value: running, icon: Clock, color: "text-blue-600" },
    { label: "Completed", value: completed, icon: CheckCircle, color: "text-green-600" },
    { label: "Failed", value: failed, icon: XCircle, color: "text-red-600" },
  ];

  return (
    <>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="mt-1 text-sm text-gray-500">
          Overview of your ForgeBox instance
        </p>
      </div>

      {/* Stats */}
      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((s) => (
          <div key={s.label} className="card flex items-center gap-4 p-5">
            <s.icon className={`h-8 w-8 ${s.color}`} />
            <div>
              <p className="text-sm text-gray-500">{s.label}</p>
              <p className="text-2xl font-semibold text-gray-900">{s.value}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Recent Tasks */}
      <div className="card mb-8">
        <div className="flex items-center justify-between border-b border-gray-200 px-5 py-4">
          <h2 className="text-lg font-semibold text-gray-900">Recent Tasks</h2>
          <Link to="/tasks" className="btn-primary text-xs">
            Run Task <ArrowRight className="ml-1 h-3 w-3" />
          </Link>
        </div>
        {recent.length === 0 ? (
          <div className="p-8 text-center text-sm text-gray-400">
            No tasks yet.{" "}
            <Link to="/tasks" className="text-forge-600 underline">
              Run a task
            </Link>{" "}
            to get started.
          </div>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-100 text-xs uppercase text-gray-500">
              <tr>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Prompt</th>
                <th className="px-5 py-3">Provider</th>
                <th className="px-5 py-3">Cost</th>
                <th className="px-5 py-3">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {recent.map((t) => (
                <tr key={t.id} className="hover:bg-gray-50">
                  <td className="px-5 py-3"><StatusBadge status={t.status} /></td>
                  <td className="max-w-xs truncate px-5 py-3 text-gray-700">
                    {t.prompt.length > 80 ? `${t.prompt.slice(0, 80)}...` : t.prompt}
                  </td>
                  <td className="px-5 py-3 text-gray-600">{t.provider}</td>
                  <td className="px-5 py-3 text-gray-600">${t.cost.toFixed(4)}</td>
                  <td className="px-5 py-3 text-gray-400">
                    {new Date(t.created_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Quick Actions */}
      <h2 className="mb-4 text-lg font-semibold text-gray-900">Quick Actions</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {[
          { label: "Run a Task", desc: "Execute an AI task in a secure VM", to: "/tasks" },
          { label: "View Sessions", desc: "Browse active and past sessions", to: "/sessions" },
          { label: "Configure Providers", desc: "Manage LLM provider settings", to: "/settings" },
        ].map((a) => (
          <Link key={a.to} to={a.to} className="card flex items-center justify-between p-5 hover:border-forge-300 transition-colors">
            <div>
              <p className="font-medium text-gray-900">{a.label}</p>
              <p className="text-sm text-gray-500">{a.desc}</p>
            </div>
            <ArrowRight className="h-5 w-5 text-gray-400" />
          </Link>
        ))}
      </div>
    </>
  );
}
