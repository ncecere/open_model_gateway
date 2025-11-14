import { useContext } from "react";
import { UserAuthContext } from "../auth/UserAuthProvider";

export function useUserAuth() {
  const ctx = useContext(UserAuthContext);
  if (!ctx) {
    throw new Error("useUserAuth must be used within a UserAuthProvider");
  }
  return ctx;
}
