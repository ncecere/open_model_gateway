import { userApi } from "../userClient";

export type DirectoryUser = {
  id: string;
  email: string;
  name: string;
};

export async function searchDirectoryUsers(query: string) {
  const { data } = await userApi.get<{ users: DirectoryUser[] }>("/directory/users", {
    params: { query },
  });
  return data.users;
}
