import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";
import {
  useUserProfileQuery,
  useUpdateUserProfileMutation,
  useChangePasswordMutation,
} from "../hooks/useUserData";
import { useTheme } from "@/providers/ThemeProvider";
import type { ThemePreference } from "@/types/theme";

interface ProfileDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UserProfileDialog({ open, onOpenChange }: ProfileDialogProps) {
  const profileQuery = useUserProfileQuery();
  const updateProfile = useUpdateUserProfileMutation();
  const changePassword = useChangePasswordMutation();
  const { toast } = useToast();
  const { setThemePreference: setGlobalTheme } = useTheme();

  const [name, setName] = useState("");
  const [themePreference, setThemePreference] =
    useState<ThemePreference>("system");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  useEffect(() => {
    if (profileQuery.data) {
      setName(profileQuery.data.name ?? "");
      if (profileQuery.data.theme_preference) {
        setThemePreference(profileQuery.data.theme_preference);
      }
    }
  }, [profileQuery.data, open]);

  const handleSave = async () => {
    try {
      await updateProfile.mutateAsync({
        name: name.trim(),
        theme_preference: themePreference,
      });
      setGlobalTheme(themePreference);
      toast({ title: "Profile updated" });
    } catch (error) {
      console.error(error);
      toast({
        variant: "destructive",
        title: "Failed to update profile",
      });
    }
  };

  const handleChangePassword = async () => {
    if (!profileQuery.data?.can_change_password) {
      return;
    }
    if (newPassword !== confirmPassword) {
      toast({
        variant: "destructive",
        title: "Passwords do not match",
      });
      return;
    }
    try {
      await changePassword.mutateAsync({
        current_password: currentPassword,
        new_password: newPassword,
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast({ title: "Password updated" });
    } catch (error) {
      console.error(error);
      toast({
        variant: "destructive",
        title: "Failed to change password",
        description: "Check your current password and try again.",
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Account profile</DialogTitle>
        </DialogHeader>
        {profileQuery.isLoading ? (
          <Skeleton className="h-40 w-full" />
        ) : profileQuery.data ? (
          <div className="space-y-8">
            <section className="space-y-4">
              <div>
                <h3 className="text-sm font-semibold">Account</h3>
                <p className="text-xs text-muted-foreground">
                  Update your display name.
                </p>
              </div>
              <div className="grid gap-4">
                <div className="space-y-2">
                  <Label>Email</Label>
                  <Input value={profileQuery.data.email} readOnly disabled />
                </div>
              <div className="space-y-2">
                <Label htmlFor="profile-name">Name</Label>
                <Input
                  id="profile-name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="theme-preference">Theme</Label>
                <Select
                  value={themePreference}
                  onValueChange={(value: ThemePreference) =>
                    setThemePreference(value)
                  }
                >
                  <SelectTrigger id="theme-preference">
                    <SelectValue placeholder="Select theme" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="system">System (match device)</SelectItem>
                    <SelectItem value="light">Light</SelectItem>
                    <SelectItem value="dark">Dark</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Choose how the interface should look.
                </p>
              </div>
              <div className="flex justify-end">
                <Button
                  onClick={handleSave}
                    disabled={updateProfile.isPending || name.trim() === ""}
                  >
                    Save changes
                  </Button>
                </div>
              </div>
            </section>

            <Separator />

            <section className="space-y-4">
              <div>
                <h3 className="text-sm font-semibold">Password</h3>
                <p className="text-xs text-muted-foreground">
                  {profileQuery.data.can_change_password
                    ? "Change your local account password."
                    : "Password changes are managed by your identity provider."}
                </p>
              </div>
              {profileQuery.data.can_change_password ? (
                <div className="grid gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="current-password">Current password</Label>
                    <Input
                      id="current-password"
                      type="password"
                      value={currentPassword}
                      onChange={(event) => setCurrentPassword(event.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-password">New password</Label>
                    <Input
                      id="new-password"
                      type="password"
                      value={newPassword}
                      onChange={(event) => setNewPassword(event.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="confirm-password">Confirm new password</Label>
                    <Input
                      id="confirm-password"
                      type="password"
                      value={confirmPassword}
                      onChange={(event) => setConfirmPassword(event.target.value)}
                    />
                  </div>
                  <div className="flex justify-end">
                    <Button
                      variant="secondary"
                      onClick={handleChangePassword}
                      disabled={changePassword.isPending}
                    >
                      Update password
                    </Button>
                  </div>
                </div>
              ) : null}
            </section>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            Unable to load profile details.
          </p>
        )}
      </DialogContent>
    </Dialog>
  );
}
