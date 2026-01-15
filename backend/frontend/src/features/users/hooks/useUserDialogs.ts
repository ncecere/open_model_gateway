import { useState } from "react";
import type { PersonalTenantRecord } from "@/api/tenants";
import { emptyUserForm, userFormFromRecord, type UserFormState } from "../types";

export interface CreateDialogState {
  open: boolean;
  setOpen: (open: boolean) => void;
  form: UserFormState;
  setForm: (form: UserFormState) => void;
  reset: () => void;
}

export interface EditDialogState {
  open: boolean;
  setOpen: (open: boolean) => void;
  record: PersonalTenantRecord | null;
  form: UserFormState;
  setForm: (form: UserFormState) => void;
  openWith: (record: PersonalTenantRecord) => void;
  close: () => void;
}

export interface DetailsDialogState {
  open: boolean;
  setOpen: (open: boolean) => void;
  record: PersonalTenantRecord | null;
  openWith: (record: PersonalTenantRecord) => void;
  close: () => void;
}

export interface DeleteDialogState {
  open: boolean;
  setOpen: (open: boolean) => void;
  records: PersonalTenantRecord[];
  openWith: (records: PersonalTenantRecord[]) => void;
  close: () => void;
}

export function useUserDialogs() {
  // Create dialog state
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState<UserFormState>(emptyUserForm);

  const createDialog: CreateDialogState = {
    open: createOpen,
    setOpen: setCreateOpen,
    form: createForm,
    setForm: setCreateForm,
    reset: () => {
      setCreateForm(emptyUserForm());
      setCreateOpen(false);
    },
  };

  // Edit dialog state
  const [editOpen, setEditOpen] = useState(false);
  const [editRecord, setEditRecord] = useState<PersonalTenantRecord | null>(null);
  const [editForm, setEditForm] = useState<UserFormState>(emptyUserForm);

  const editDialog: EditDialogState = {
    open: editOpen,
    setOpen: setEditOpen,
    record: editRecord,
    form: editForm,
    setForm: setEditForm,
    openWith: (record: PersonalTenantRecord) => {
      setEditRecord(record);
      setEditForm(userFormFromRecord(record));
      setEditOpen(true);
    },
    close: () => {
      setEditOpen(false);
      setEditRecord(null);
      setEditForm(emptyUserForm());
    },
  };

  // Details dialog state
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [detailsRecord, setDetailsRecord] = useState<PersonalTenantRecord | null>(null);

  const detailsDialog: DetailsDialogState = {
    open: detailsOpen,
    setOpen: setDetailsOpen,
    record: detailsRecord,
    openWith: (record: PersonalTenantRecord) => {
      setDetailsRecord(record);
      setDetailsOpen(true);
    },
    close: () => {
      setDetailsOpen(false);
      setDetailsRecord(null);
    },
  };

  // Delete confirmation dialog state
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteRecords, setDeleteRecords] = useState<PersonalTenantRecord[]>([]);

  const deleteDialog: DeleteDialogState = {
    open: deleteOpen,
    setOpen: setDeleteOpen,
    records: deleteRecords,
    openWith: (records: PersonalTenantRecord[]) => {
      setDeleteRecords(records);
      setDeleteOpen(true);
    },
    close: () => {
      setDeleteOpen(false);
      setDeleteRecords([]);
    },
  };

  // Clone handler - opens create dialog pre-filled
  const handleClone = (record: PersonalTenantRecord) => {
    setCreateForm({
      ...userFormFromRecord(record),
      email: "", // Must be unique
      password: "",
      sendInvite: true,
    });
    setCreateOpen(true);
  };

  return {
    createDialog,
    editDialog,
    detailsDialog,
    deleteDialog,
    handleClone,
  };
}
