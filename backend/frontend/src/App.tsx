import { AdminApp } from "./apps/admin/App";
import { UserApp } from "./apps/user/App";

export function App() {
  const pathname =
    typeof window !== "undefined" ? window.location.pathname : "";
  const isAdminPortal = pathname.startsWith("/admin");
  return isAdminPortal ? <AdminApp /> : <UserApp />;
}

export default App;
