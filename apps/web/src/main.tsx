import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { DesignSystemProvider } from "@heph/design-system";
import "@heph/design-system/styles.css";
import { App } from "./App";

const root = document.getElementById("root");
if (!root) throw new Error("Missing application root");

createRoot(root).render(
  <StrictMode>
    <DesignSystemProvider>
      <App />
    </DesignSystemProvider>
  </StrictMode>,
);
