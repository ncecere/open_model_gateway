import { useEffect } from "react";
import { AdminApp } from "./apps/admin/App";
import { UserApp } from "./apps/user/App";
import LogoMark from "@/assets/system/open_model_gateway.svg";

export function App() {
  useEffect(() => {
    if (typeof document === "undefined") {
      return;
    }
    const createIcon = (rel: string) => {
      const link = document.createElement("link");
      link.setAttribute("rel", rel);
      document.head.appendChild(link);
      return link;
    };

    const ensureIcon = (rel: string) => {
      const existing = document.head.querySelector<HTMLLinkElement>(
        `link[rel='${rel}']`,
      );
      return existing ?? createIcon(rel);
    };

    const iconLinks = [ensureIcon("icon"), ensureIcon("shortcut icon")];
    iconLinks.forEach((link) => {
      link.type = "image/svg+xml";
      link.href = LogoMark;
    });
  }, []);

  const pathname =
    typeof window !== "undefined" ? window.location.pathname : "";
  const isAdminPortal = pathname.startsWith("/admin");
  return isAdminPortal ? <AdminApp /> : <UserApp />;
}

export default App;
