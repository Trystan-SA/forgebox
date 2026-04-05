import { clsx } from "clsx";

const providerColors: Record<string, string> = {
  anthropic: "bg-amber-100 text-amber-800",
  openai: "bg-emerald-100 text-emerald-800",
  google: "bg-blue-100 text-blue-800",
  ollama: "bg-purple-100 text-purple-800",
  bedrock: "bg-orange-100 text-orange-800",
  vertex: "bg-cyan-100 text-cyan-800",
  openrouter: "bg-pink-100 text-pink-800",
};

export function ProviderBadge({ name }: { name: string }) {
  const color = providerColors[name.toLowerCase()] ?? "bg-gray-100 text-gray-700";

  return (
    <span className={clsx("badge", color)}>
      {name}
    </span>
  );
}
