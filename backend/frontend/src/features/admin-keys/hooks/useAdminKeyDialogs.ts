import { useState } from "react";
import type { AdminKeyRecord } from "../types";
import { emptyAdminKeyForm, type AdminKeyFormState } from "../types";

export interface CreateDialogState {
  open: boolean;
  setOpen: (open: boolean) => void;
  form: AdminKeyFormState;
  setForm: (form: AdminKeyFormState) => void;
  reset: () => void;
}

export interface DetailsDialogState {
  open: boolean;
  setOpen: (open: boolean) => void;
  record: AdminKeyRecord | null;
  openWith: (record: AdminKeyRecord) => void;
  close: () => void;
}

export interface IssuedDialogState {
  open: boolean;
  token: string | null;
  setToken: (token: string | null) => void;
  close: () => void;
}

export interface RevokeDialogState {
  open: boolean;
  records: AdminKeyRecord[];
  openWith: (records: AdminKeyRecord[]) => void;
  close: () => void;
}

export function useAdminKeyDialogs() {
  // Create dialog state
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState<AdminKeyFormState>(emptyAdminKeyForm());

  const createDialog: CreateDialogState = {
    open: createOpen,
    setOpen: setCreateOpen,
    form: createForm,
    setForm: setCreateForm,
    reset: () => {
      setCreateForm(emptyAdminKeyForm());
      setCreateOpen(false);
    },
  };

  // Details dialog state
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [detailsRecord, setDetailsRecord] = useState<AdminKeyRecord | null>(null);

  const detailsDialog: DetailsDialogState = {
    open: detailsOpen,
    setOpen: setDetailsOpen,
    record: detailsRecord,
    openWith: (record: AdminKeyRecord) => {
      setDetailsRecord(record);
      setDetailsOpen(true);
    },
    close: () => {
      setDetailsOpen(false);
      setDetailsRecord(null);
    },
  };

  // Issued token dialog state
  const [issuedToken, setIssuedToken] = useState<string | null>(null);

  const issuedDialog: IssuedDialogState = {
    open: issuedToken !== null,
    token: issuedToken,
    setToken: setIssuedToken,
    close: () => setIssuedToken(null),
  };

  // Revoke confirmation dialog state
  const [revokeOpen, setRevokeOpen] = useState(false);
  const [revokeRecords, setRevokeRecords] = useState<AdminKeyRecord[]>([]);

  const revokeDialog: RevokeDialogState = {
    open: revokeOpen,
    records: revokeRecords,
    openWith: (records: AdminKeyRecord[]) => {
      setRevokeRecords(records);
      setRevokeOpen(true);
    },
    close: () => {
      setRevokeOpen(false);
      setRevokeRecords([]);
    },
  };

  return {
    createDialog,
    detailsDialog,
    issuedDialog,
    revokeDialog,
  };
}
