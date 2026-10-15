import type { ReactNode } from "react";
import { Theme } from "@astryxdesign/core/theme";
import { neutralTheme } from "@astryxdesign/theme-neutral/built";

type Children = { children: ReactNode };

export function DesignSystemProvider({ children }: Children) {
  return <Theme theme={neutralTheme}>{children}</Theme>;
}

export function AppFrame({ children }: Children) {
  return (
    <div className="heph-app">
      <header className="heph-header">
        <a href="/" className="heph-brand">
          heph<span className="heph-version">v2</span>
        </a>
        <span className="heph-header-label">Git forge + agent runtime</span>
      </header>
      <main className="heph-main">{children}</main>
    </div>
  );
}

export function Panel({ children }: Children) {
  return <section className="heph-panel">{children}</section>;
}

export function Heading({ children }: Children) {
  return <h1 className="heph-heading">{children}</h1>;
}

export function Text({ children }: Children) {
  return <p className="heph-text">{children}</p>;
}

export function Stack({ children }: Children) {
  return <div className="heph-stack">{children}</div>;
}

export function Status({ children, tone }: Children & { tone: "neutral" | "success" | "error" }) {
  return (
    <p className={`heph-status heph-status-${tone}`} role="status" aria-live="polite">
      {children}
    </p>
  );
}
