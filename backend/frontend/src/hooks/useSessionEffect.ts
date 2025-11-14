import { useEffect } from "react";
import { useAuth } from "./useAuth";

export function useSessionEffect() {
  const { refresh } = useAuth();

  useEffect(() => {
    const interval = window.setInterval(
      () => {
        refresh().catch(() => {
          // swallow error; useAuth handles clearing session
        });
      },
      15 * 60 * 1000,
    );

    return () => {
      window.clearInterval(interval);
    };
  }, [refresh]);
}
