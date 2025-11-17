import { createHttpClient, type RequestConfig } from "./httpClient";

export const USER_SKIP_AUTH_KEY = "__skipUserRefresh";

const {
  instance: userApi,
  setAuthToken: setUserAuthToken,
  setUnauthorizedHandler: setUserUnauthorizedHandler,
} = createHttpClient({
  baseURL: "/user",
  skipAuthKey: USER_SKIP_AUTH_KEY,
});

export { userApi, setUserAuthToken, setUserUnauthorizedHandler };

export type UserRequestConfig<D = unknown> = RequestConfig<D>;
