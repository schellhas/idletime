import { useAuth } from "../auth/AuthContext";
import { ChangePasswordForm } from "./ChangePasswordForm";
import { DataSection } from "./DataSection";
import { ProfileForm } from "./ProfileForm";

export function AccountPage() {
  const { logout } = useAuth();

  return (
    <div className="stack stack--wide account">
      <div className="page-head">
        <h1>Account</h1>
        {/* On phones the tab bar has no room for it, so logging out lives here too. */}
        <button type="button" className="button button--secondary button--small" onClick={() => void logout()}>
          Log out
        </button>
      </div>
      <ProfileForm />
      <ChangePasswordForm />
      <DataSection />
    </div>
  );
}
