import { Toaster as SonnerToaster, type ToasterProps } from "sonner";

const toastOptions: ToasterProps["toastOptions"] = {
  classNames: {
    toast:
      "group toast bg-background text-foreground border border-border shadow-lg",
    description: "group-[.toast]:text-muted-foreground",
    actionButton:
      "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
    cancelButton:
      "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
  },
};

const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <SonnerToaster
      position="top-right"
      theme="system"
      toastOptions={toastOptions}
      {...props}
    />
  );
};

export { Toaster };
