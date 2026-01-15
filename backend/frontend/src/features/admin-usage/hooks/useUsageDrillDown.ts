import { useState, useCallback, useMemo } from "react";

export type DrillDownLevel = "overview" | "tenant" | "user" | "key" | "model";

export interface DrillDownState {
  level: DrillDownLevel;
  tenantId?: string;
  tenantName?: string;
  userId?: string;
  userName?: string;
  keyId?: string;
  keyName?: string;
  modelAlias?: string;
}

export interface BreadcrumbItem {
  level: DrillDownLevel;
  label: string;
  state: DrillDownState;
}

export function useUsageDrillDown() {
  const [state, setState] = useState<DrillDownState>({ level: "overview" });

  const drillIntoTenant = useCallback((tenantId: string, tenantName?: string) => {
    setState({
      level: "tenant",
      tenantId,
      tenantName,
    });
  }, []);

  const drillIntoUser = useCallback(
    (userId: string, userName?: string) => {
      setState((prev) => ({
        ...prev,
        level: "user",
        userId,
        userName,
      }));
    },
    []
  );

  const drillIntoKey = useCallback(
    (keyId: string, keyName?: string) => {
      setState((prev) => ({
        ...prev,
        level: "key",
        keyId,
        keyName,
      }));
    },
    []
  );

  const drillIntoModel = useCallback((modelAlias: string) => {
    setState({
      level: "model",
      modelAlias,
    });
  }, []);

  const goToLevel = useCallback((level: DrillDownLevel) => {
    setState((prev) => {
      switch (level) {
        case "overview":
          return { level: "overview" };
        case "tenant":
          return {
            level: "tenant",
            tenantId: prev.tenantId,
            tenantName: prev.tenantName,
          };
        case "user":
          return {
            level: "user",
            tenantId: prev.tenantId,
            tenantName: prev.tenantName,
            userId: prev.userId,
            userName: prev.userName,
          };
        case "key":
          return prev;
        case "model":
          return {
            level: "model",
            modelAlias: prev.modelAlias,
          };
        default:
          return prev;
      }
    });
  }, []);

  const goBack = useCallback(() => {
    setState((prev) => {
      switch (prev.level) {
        case "key":
          return {
            level: "user",
            tenantId: prev.tenantId,
            tenantName: prev.tenantName,
            userId: prev.userId,
            userName: prev.userName,
          };
        case "user":
          return {
            level: "tenant",
            tenantId: prev.tenantId,
            tenantName: prev.tenantName,
          };
        case "tenant":
        case "model":
          return { level: "overview" };
        default:
          return prev;
      }
    });
  }, []);

  const reset = useCallback(() => {
    setState({ level: "overview" });
  }, []);

  const breadcrumbs = useMemo<BreadcrumbItem[]>(() => {
    const items: BreadcrumbItem[] = [
      {
        level: "overview",
        label: "Overview",
        state: { level: "overview" },
      },
    ];

    if (state.level === "model" && state.modelAlias) {
      items.push({
        level: "model",
        label: state.modelAlias,
        state: { level: "model", modelAlias: state.modelAlias },
      });
      return items;
    }

    if (state.tenantId) {
      items.push({
        level: "tenant",
        label: state.tenantName ?? state.tenantId,
        state: {
          level: "tenant",
          tenantId: state.tenantId,
          tenantName: state.tenantName,
        },
      });
    }

    if (state.userId) {
      items.push({
        level: "user",
        label: state.userName ?? state.userId,
        state: {
          level: "user",
          tenantId: state.tenantId,
          tenantName: state.tenantName,
          userId: state.userId,
          userName: state.userName,
        },
      });
    }

    if (state.keyId) {
      items.push({
        level: "key",
        label: state.keyName ?? state.keyId,
        state: state,
      });
    }

    return items;
  }, [state]);

  const canGoBack = state.level !== "overview";

  return {
    state,
    breadcrumbs,
    canGoBack,
    drillIntoTenant,
    drillIntoUser,
    drillIntoKey,
    drillIntoModel,
    goToLevel,
    goBack,
    reset,
  };
}
