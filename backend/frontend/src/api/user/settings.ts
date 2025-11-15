import { userApi } from "../userClient";

export interface UserFileSettings {
  default_ttl_seconds: number;
  max_ttl_seconds: number;
}

export async function getUserFileSettings() {
  const { data } = await userApi.get<UserFileSettings>("/files/settings");
  return data;
}
