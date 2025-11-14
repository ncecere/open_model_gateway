import { userApi } from "../userClient";

export interface UserModel {
  alias: string;
  provider: string;
  enabled: boolean;
}

export async function listUserModels(): Promise<UserModel[]> {
  const { data } = await userApi.get<{ models: UserModel[] }>("/models");
  return data.models;
}
