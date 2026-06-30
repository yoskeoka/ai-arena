import { FormEvent, useMemo, useState } from "react";

import { OperatorApiClient, SignupInviteResponse, SignupInviteRole } from "../../api";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, messageOf, normalizeBaseUrl } from "./operatorPageSupport";

type InvitesPageProps = {
  baseUrl: string;
};

const roleOptions: Array<{ value: SignupInviteRole; label: string; description: string }> = [
  { value: "participant", label: "Participant", description: "Default external player access." },
  { value: "developer", label: "Developer", description: "Collaborator access for build and operator work." },
  { value: "operator", label: "Operator", description: "Full operator access for the private surface." },
];

export function InvitesPage({ baseUrl }: InvitesPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [role, setRole] = useState<SignupInviteRole>("operator");
  const [ttl, setTtl] = useState("24h");
  const [writeState, setWriteState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [writeError, setWriteError] = useState<string>();
  const [invite, setInvite] = useState<SignupInviteResponse>();

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setWriteState("submitting");
    setWriteError(undefined);
    try {
      const payload: { role: SignupInviteRole; ttl?: string } = { role };
      const trimmedTTL = ttl.trim();
      if (trimmedTTL !== "") {
        payload.ttl = trimmedTTL;
      }
      const response = await client.createSignupInvite(payload);
      setInvite(response);
      setWriteState("success");
    } catch (error) {
      setWriteState("error");
      setWriteError(messageOf(error));
    }
  };

  return (
    <section className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
      <Panel
        title="Issue Signup Invite"
        subtitle="Create one invite token for participant, developer, or operator bootstrap."
        status={writeState}
        error={writeError}
        hint={hintFor(writeError)}
        testId="operator-form-invites"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <label className="flex flex-col gap-2 text-sm">
            <span className="font-medium text-black/70">Role</span>
            <select
              className="rounded-2xl border border-black/15 bg-white px-4 py-3 shadow-sm outline-none transition focus:border-accent"
              value={role}
              onChange={(event) => setRole(event.target.value as SignupInviteRole)}
            >
              {roleOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <span className="text-xs text-black/55">
              {roleOptions.find((option) => option.value === role)?.description ?? "Select the invite target role."}
            </span>
          </label>
          <label className="flex flex-col gap-2 text-sm">
            <span className="font-medium text-black/70">TTL</span>
            <input
              className="rounded-2xl border border-black/15 bg-white px-4 py-3 shadow-sm outline-none transition focus:border-accent"
              value={ttl}
              onChange={(event) => setTtl(event.target.value)}
              placeholder="24h"
            />
            <span className="text-xs text-black/55">Leave blank to use the backend default invite lifetime.</span>
          </label>
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit">
            Create invite
          </button>
        </form>

        {invite ? (
          <div className="mt-6 rounded-3xl border border-black/10 bg-paper p-4" data-testid="signup-invite-result">
            <p className="text-sm font-semibold uppercase tracking-[0.16em] text-black/55">Latest Invite</p>
            <dl className="mt-3 grid gap-3 text-sm">
              <ResultField label="Role" value={invite.role} testId="signup-invite-role" />
              <ResultField label="Invite Token" value={invite.invite_token} monospace testId="signup-invite-token" />
              <ResultField label="Expires At" value={invite.expires_at} monospace testId="signup-invite-expires-at" />
              <div className="grid gap-1">
                <dt className="font-medium text-black/70">Invite URL</dt>
                <dd className="break-all">
                  <a className="font-mono text-sm text-teal no-underline hover:text-ink" href={invite.invite_url} data-testid="signup-invite-url">
                    {invite.invite_url}
                  </a>
                </dd>
              </div>
            </dl>
          </div>
        ) : (
          <p className="mt-6 text-sm text-black/60">Create an invite to reveal the token and login URL.</p>
        )}
      </Panel>
    </section>
  );
}

function ResultField({
  label,
  value,
  monospace,
  testId,
}: {
  label: string;
  value: string;
  monospace?: boolean;
  testId: string;
}) {
  return (
    <div className="grid gap-1">
      <dt className="font-medium text-black/70">{label}</dt>
      <dd className={monospace ? "font-mono break-all text-sm" : "break-all"} data-testid={testId}>
        {value}
      </dd>
    </div>
  );
}
