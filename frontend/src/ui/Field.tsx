import type { InputHTMLAttributes } from "react";

interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  hint?: string;
}

export function Field({ label, hint, ...input }: FieldProps) {
  return (
    <label className="field">
      <span>{label}</span>
      <input {...input} />
      {hint && <small className="field__hint">{hint}</small>}
    </label>
  );
}
