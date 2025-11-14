import type { ThemePreference } from "@/types/theme";
import { userApi } from "../userClient";
import type { UserProfile } from "../types";

export async function getUserProfile() {
  const { data } = await userApi.get<UserProfile>("/profile");
  return data;
}

export type UpdateUserProfileRequest = {
  name?: string;
  theme_preference?: ThemePreference;
};

export async function updateUserProfile(payload: UpdateUserProfileRequest) {
  const { data } = await userApi.patch<UserProfile>("/profile", payload);
  return data;
}

export type ChangeUserPasswordRequest = {
  current_password: string;
  new_password: string;
};

export async function changeUserPassword(payload: ChangeUserPasswordRequest) {
  const { data } = await userApi.post<{ success: boolean }>(
    "/profile/password",
    payload,
  );
  return data;
}
