import { describe, expect, it } from "vitest";

import { MAX_SLOT_MINUTES, parseSlotMinutes } from "./slot";

describe("parseSlotMinutes", () => {
  it.each([
    ["25", 25],
    [" 90 ", 90],
    ["1", 1],
    [String(MAX_SLOT_MINUTES), MAX_SLOT_MINUTES],
  ])("accepts %j", (input, expected) => {
    expect(parseSlotMinutes(input)).toBe(expected);
  });

  it.each([[""], ["0"], ["-5"], ["2.5"], ["abc"], ["1e3"], [String(MAX_SLOT_MINUTES + 1)]])("rejects %j", (input) => {
    expect(parseSlotMinutes(input)).toBeNull();
  });
});
