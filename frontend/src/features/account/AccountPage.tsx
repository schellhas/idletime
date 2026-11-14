import { ChangePasswordForm } from "./ChangePasswordForm";
import { DataSection } from "./DataSection";
import { ProfileForm } from "./ProfileForm";

export function AccountPage() {
  return (
    <div className="stack stack--wide account">
      <h1>Account</h1>
      <ProfileForm />
      <ChangePasswordForm />
      <DataSection />
    </div>
  );
}
