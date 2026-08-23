import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App.jsx";
import "./design-system/index.css";
import "./assets/styles.css";
import { ThemeProvider } from "./context/ThemeContext.jsx";

createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <ThemeProvider>
      <App />
    </ThemeProvider>
  </React.StrictMode>
);

