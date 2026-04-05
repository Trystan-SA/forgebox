import { useEffect, useState } from "react";
import { format } from "date-fns";
import { Shield, Search } from "lucide-react";
import { clsx } from "clsx";
import { listAuditEntries } from "../lib/api";
import type { AuditEntry } from "../lib/types";

export function AuditLog() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [userId, setUserId] = useState("");
  const [decision, setDecision] = useState<"all" | "allow" | "deny">("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  useEffect(() => {
    listAuditEntries()
      .then(setEntries)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const filtered = entries.filter((e) => {
    if (userId && !e.user_id.toLowerCase().includes(userId.toLowerCase())) return false;
    if (decision !== "all" && e.decision !== decision) return false;
    if (dateFrom && e.timestamp < dateFrom) return false;
    if (dateTo && e.timestamp > dateTo) return false;
    return true;
  });

  return (
    <>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Audit Log</h1>
        <p className="mt-1 text-sm text-gray-500">
          Track all tool calls and permission decisions.
        </p>
      </div>

      {/* Filters */}
      <div className="mb-6 flex flex-wrap items-end gap-3">
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-600">User ID</label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-gray-400" />
            <input className="input pl-8 w-48" placeholder="Filter by user..." value={userId} onChange={(e) => setUserId(e.target.value)} />
          </div>
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-600">Decision</label>
          <select className="input w-32" value={decision} onChange={(e) => setDecision(e.target.value as "all" | "allow" | "deny")}>
            <option value="all">All</option>
            <option value="allow">Allow</option>
            <option value="deny">Deny</option>
          </select>
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-600">From</label>
          <input className="input w-40" type="text" placeholder="YYYY-MM-DD" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} />
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-600">To</label>
          <input className="input w-40" type="text" placeholder="YYYY-MM-DD" value={dateTo} onChange={(e) => setDateTo(e.target.value)} />
        </div>
      </div>

      {loading && <p className="text-gray-500">Loading...</p>}
      {error && <p className="text-red-600">Error: {error}</p>}

      {!loading && !error && filtered.length === 0 && (
        <div className="card flex flex-col items-center justify-center p-12 text-center text-gray-400">
          <Shield className="mb-3 h-10 w-10" />
          <p className="text-sm">No audit entries found.</p>
        </div>
      )}

      {!loading && !error && filtered.length > 0 && (
        <div className="card overflow-hidden">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-100 bg-gray-50 text-xs uppercase text-gray-500">
              <tr>
                <th className="px-5 py-3">Timestamp</th>
                <th className="px-5 py-3">User</th>
                <th className="px-5 py-3">Action</th>
                <th className="px-5 py-3">Tool</th>
                <th className="px-5 py-3">Decision</th>
                <th className="px-5 py-3">Reason</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {filtered.map((entry) => (
                <tr key={entry.id} className="hover:bg-gray-50">
                  <td className="whitespace-nowrap px-5 py-3 text-gray-400">
                    {format(new Date(entry.timestamp), "MMM d, yyyy HH:mm:ss")}
                  </td>
                  <td className="px-5 py-3 font-mono text-xs text-gray-700">
                    {entry.user_id.slice(0, 8)}
                  </td>
                  <td className="px-5 py-3 text-gray-600">{entry.action}</td>
                  <td className="px-5 py-3 font-mono text-xs text-gray-600">{entry.tool ?? "-"}</td>
                  <td className="px-5 py-3">
                    <span className={clsx(
                      "badge",
                      entry.decision === "allow" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700",
                    )}>
                      {entry.decision}
                    </span>
                  </td>
                  <td className="max-w-xs truncate px-5 py-3 text-gray-500">
                    {entry.reason ?? "-"}
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
