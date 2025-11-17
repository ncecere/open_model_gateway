import React from "react";
import ReactDOM from "react-dom/client";

import { UserApp } from "./App";
import LogoMark from "@/assets/system/open_model_gateway.svg";
import { applyBranding } from "@/lib/branding";

import "@/index.css";

applyBranding(LogoMark);

ReactDOM.createRoot(document.getElementById("app") as HTMLElement).render(
  <React.StrictMode>
    <UserApp />
  </React.StrictMode>,
);
