import "@mantine/core/styles.css";
import "@mantine/spotlight/styles.css";
import "./app.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { MantineProvider, createTheme } from "@mantine/core";
import { App } from "./App";

const theme = createTheme({
  primaryColor: "blue",
  defaultRadius: "sm",
  fontFamilyMonospace: "ui-monospace, SFMono-Regular, Menlo, monospace",
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="auto">
      <App />
    </MantineProvider>
  </StrictMode>,
);
