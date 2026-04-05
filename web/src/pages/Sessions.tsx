import { useEffect, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { MessageSquare } from "lucide-react";
import { listSessions } from "../lib/api";
import type { Session } from "../lib/types";

export function Sessions() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listSessions()
      .then(setSessions)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Sessions</h1>
        <p className="mt-1 text-sm text-gray-500">
          Browse active and past provider sessions.
        </p>
      </div>

      {loading && <p className="text-gray-500">Loading...</p>}
      {error && <p className="text-red-600">Error: {error}</p>}

      {!loading && !error && sessions.length === 0 && (
        <div className="card flex flex-col items-center justify-center p-12 text-center text-gray-400">
          <MessageSquare className="mb-3 h-10 w-10" />
          <p className="text-sm">No sessions yet. Run a task to start.</p>
        </div>
      )}

      {!loading && !error && sessions.length > 0 && (
        <div className="card overflow-hidden">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-100 bg-gray-50 text-xs uppercase text-gray-500">
              <tr>
                <th className="px-5 py-3">ID</th>
                <th className="px-5 py-3">Provider</th>
                <th className="px-5 py-3">Model</th>
                <th className="px-5 py-3">Created</th>
                <th className="px-5 py-3">Last Active</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {sessions.map((s) => (
                <tr key={s.id} className="hover:bg-gray-50">
                  <td className="px-5 py-3 font-mono text-xs text-gray-700">
                    {s.id.slice(0, 8)}
                  </td>
                  <td className="px-5 py-3 text-gray-600">{s.provider}</td>
                  <td className="px-5 py-3 text-gray-600">{s.model}</td>
                  <td className="px-5 py-3 text-gray-400">
                    {formatDistanceToNow(new Date(s.created_at), { addSuffix: true })}
                  </td>
                  <td className="px-5 py-3 text-gray-400">
                    {formatDistanceToNow(new Date(s.updated_at), { addSuffix: true })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
