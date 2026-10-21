import type { Description, State } from "./api";

export const readable = new Set(["sensor", "text_sensor", "binary_sensor"]);

export function diagnostic(d: Description): boolean {
  return d.info.entity_category === "ENTITY_CATEGORY_DIAGNOSTIC";
}

export function shown(d: Description, state: State["state"] | undefined): string {
  if (!state || state.missing_state === true) return "—";
  const v = state.state;
  if (typeof v === "boolean") return v ? "On" : "Off";
  if (typeof v === "number") {
    const places = d.info.accuracy_decimals ?? (Number.isInteger(v) ? 0 : 1);
    return v.toFixed(places);
  }
  if (typeof v === "string") return v || "—";
  return JSON.stringify(state);
}
