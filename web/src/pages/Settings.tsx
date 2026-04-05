import { useEffect, useState } from "react";
import { clsx } from "clsx";
import { Plug, Radio, Sliders, Plus, CheckCircle, XCircle } from "lucide-react";
import { listProviders } from "../lib/api";
import type { Provider } from "../lib/types";

type Tab = "providers" | "channels" | "general";

const tabs: { key: Tab; label: string; icon: typeof Plug }[] = [
  { key: "providers", label: "Providers", icon: Plug },
  { key: "channels", label: "Channels", icon: Radio },
  { key: "general", label: "General", icon: Sliders },
];

const defaultChannels = [
  { name: "Slack", configured: false },
  { name: "Discord", configured: false },
  { name: "Webhook", configured: true },
  { name: "Email", configured: false },
];

export function Settings() {
  const [activeTab, setActiveTab] = useState<Tab>("providers");
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listProviders()
      .then(setProviders)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
      </div>

      {/* Tabs */}
      <div className="mb-6 flex gap-1 rounded-lg bg-gray-100 p-1">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={clsx(
              "flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors",
              activeTab === tab.key
                ? "bg-white text-gray-900 shadow-sm"
                : "text-gray-500 hover:text-gray-700",
            )}
          >
            <tab.icon className="h-4 w-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {loading && <p className="text-gray-500">Loading...</p>}
      {error && <p className="text-red-600">Error: {error}</p>}

      {/* Providers */}
      {!loading && !error && activeTab === "providers" && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Configured Providers</h2>
            <button className="btn-secondary text-xs">
              <Plus className="mr-1 h-3 w-3" /> Add Provider
            </button>
          </div>
          {providers.length === 0 ? (
            <p className="text-sm text-gray-400">No providers configured.</p>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {providers.map((p) => (
                <div key={p.name} className="card flex items-center justify-between p-5">
                  <div>
                    <p className="font-medium text-gray-900">{p.name}</p>
                    <p className="text-xs text-gray-500">v{p.version} {p.builtin ? "(built-in)" : ""}</p>
                  </div>
                  <span className="badge gap-1 bg-green-100 text-green-700">
                    <CheckCircle className="h-3 w-3" /> Active
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Channels */}
      {!loading && !error && activeTab === "channels" && (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Input Channels</h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {defaultChannels.map((ch) => (
              <div key={ch.name} className="card flex items-center justify-between p-5">
                <div>
                  <p className="font-medium text-gray-900">{ch.name}</p>
                  <p className="text-xs text-gray-500">{ch.configured ? "Configured" : "Not configured"}</p>
                </div>
                {ch.configured ? (
                  <span className="badge gap-1 bg-green-100 text-green-700">
                    <CheckCircle className="h-3 w-3" /> Active
                  </span>
                ) : (
                  <span className="badge gap-1 bg-gray-100 text-gray-500">
                    <XCircle className="h-3 w-3" /> Inactive
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* General */}
      {!loading && !error && activeTab === "general" && (
        <div className="space-y-6">
          <div>
            <h2 className="mb-4 text-lg font-semibold text-gray-900">VM Defaults</h2>
            <div className="card divide-y divide-gray-100">
              {[
                { label: "Memory", value: "512 MB" },
                { label: "vCPUs", value: "1" },
                { label: "Timeout", value: "5 minutes" },
                { label: "Network Access", value: "Disabled" },
              ].map((item) => (
                <div key={item.label} className="flex items-center justify-between px-5 py-3 text-sm">
                  <span className="text-gray-600">{item.label}</span>
                  <span className="font-medium text-gray-900">{item.value}</span>
                </div>
              ))}
            </div>
          </div>
          <div>
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Storage</h2>
            <div className="card px-5 py-4 text-sm text-gray-600">
              Local filesystem storage is active. Configure S3 or GCS backends in the
              ForgeBox configuration file.
            </div>
          </div>
        </div>
      )}
    </>
  );
}
