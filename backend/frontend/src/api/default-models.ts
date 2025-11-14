import { api } from "./client";

export type DefaultModelsResponse = {
  models: string[];
};

export async function listDefaultModels() {
  const { data } = await api.get<DefaultModelsResponse>(
    "/settings/default-models",
  );
  return data.models;
}

export async function addDefaultModel(alias: string) {
  await api.post("/settings/default-models", { alias });
}

export async function removeDefaultModel(alias: string) {
  await api.delete(`/settings/default-models/${alias}`);
}
