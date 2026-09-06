import { Activity, AlertTriangle, LayoutGrid, Settings } from "lucide-react";
import { motion } from "motion/react";
import { NavLink } from "react-router-dom";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/", label: "Monitors", icon: LayoutGrid, end: true },
  { to: "/incidents", label: "Incidents", icon: AlertTriangle, end: false },
  { to: "/settings", label: "Settings", icon: Settings, end: false },
];

export function Sidebar() {
  return (
    <aside className="flex h-screen w-16 shrink-0 flex-col border-r border-border bg-surface md:w-56">
      <div className="flex h-14 items-center gap-2.5 border-b border-border px-4 md:px-5">
        <Activity className="size-5 shrink-0 text-accent" />
        <span className="hidden text-base font-semibold tracking-tight md:inline">
          PulseWatch
        </span>
      </div>

      <nav className="flex flex-1 flex-col gap-1 p-2 md:p-3">
        {navItems.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              cn(
                "group relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
                isActive ? "text-accent" : "text-fg-muted hover:bg-surface-2 hover:text-fg",
              )
            }
          >
            {({ isActive }) => (
              <>
                {/* A single shared-layout element slides smoothly between
                    whichever nav item is active, rather than each item
                    independently fading its own highlight in/out. */}
                {isActive && (
                  <motion.span
                    layoutId="sidebar-active-pill"
                    className="absolute inset-0 rounded-lg bg-accent/10"
                    transition={{ type: "spring", stiffness: 500, damping: 35 }}
                  />
                )}
                {isActive && (
                  <motion.span
                    layoutId="sidebar-active-bar"
                    className="absolute top-1/2 left-0 h-4 w-0.5 -translate-y-1/2 rounded-full bg-accent"
                    transition={{ type: "spring", stiffness: 500, damping: 35 }}
                    aria-hidden="true"
                  />
                )}
                <Icon className="relative z-10 size-[18px] shrink-0 transition-transform duration-150 group-hover:scale-110" />
                <span className="relative z-10 hidden md:inline">{label}</span>
              </>
            )}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
