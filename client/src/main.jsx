import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App.jsx";
import "./design-system/index.css";
import "./assets/styles.css";
import { ThemeProvider } from "./context/ThemeContext.jsx";
import AppErrorBoundary from "./components/AppErrorBoundary.jsx";

createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <AppErrorBoundary><ThemeProvider><App /></ThemeProvider></AppErrorBoundary>
  </React.StrictMode>
);
