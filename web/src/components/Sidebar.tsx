import { NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Play,
  MessageSquare,
  Settings,
  Shield,
  Box,
} from "lucide-react";
import { clsx } from "clsx";

const navigation = [
  { name: "Dashboard", to: "/", icon: LayoutDashboard },
  { name: "Run Task", to: "/tasks", icon: Play },
  { name: "Sessions", to: "/sessions", icon: MessageSquare },
  { name: "Settings", to: "/settings", icon: Settings },
  { name: "Audit Log", to: "/audit", icon: Shield },
];

export function Sidebar() {
  return (
    <div className="flex w-64 flex-col border-r border-gray-200 bg-white">
      {/* Logo */}
      <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-6">
        <Box className="h-7 w-7 text-forge-600" />
        <span className="text-lg font-bold text-gray-900">ForgeBox</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 px-3 py-4">
        {navigation.map((item) => (
          <NavLink
            key={item.name}
            to={item.to}
            end={item.to === "/"}
            className={({ isActive }) =>
              clsx(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-forge-50 text-forge-700"
                  : "text-gray-600 hover:bg-gray-50 hover:text-gray-900",
              )
            }
          >
            <item.icon className="h-5 w-5 shrink-0" />
            {item.name}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t border-gray-200 p-4">
        <div className="flex items-center gap-2">
          <div className="h-2 w-2 rounded-full bg-green-500" />
          <span className="text-xs text-gray-500">Gateway connected</span>
        </div>
        <p className="mt-1 text-xs text-gray-400">ForgeBox v0.1.0</p>
      </div>
    </div>
  );
}
