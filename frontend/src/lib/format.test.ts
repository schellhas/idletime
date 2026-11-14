import { describe, expect, it } from "vitest";

import { formatClock, formatMinutes, formatMinutesShort, formatPercent } from "./format";

describe("formatMinutes", () => {
  it.each([
    [0, "0 min"],
    [45, "45 min"],
    [60, "1 h"],
    [95, "1 h 35 min"],
    [120, "2 h"],
    [59.6, "1 h"],
  ])("%s -> %s", (input, expected) => {
    expect(formatMinutes(input)).toBe(expected);
  });
});

describe("formatMinutesShort", () => {
  it.each([
    [0, "0m"],
    [30, "30m"],
    [90, "1.5h"],
    [120, "2h"],
  ])("%s -> %s", (input, expected) => {
    expect(formatMinutesShort(input)).toBe(expected);
  });
});

describe("formatClock", () => {
  it.each([
    [0, "0:00"],
    [75, "1:15"],
    [3725, "1:02:05"],
    [-5, "0:00"],
  ])("%s -> %s", (input, expected) => {
    expect(formatClock(input)).toBe(expected);
  });
});

describe("formatPercent", () => {
  it("rounds to whole percent", () => {
    expect(formatPercent(0.456)).toBe("46%");
    expect(formatPercent(0)).toBe("0%");
    expect(formatPercent(1)).toBe("100%");
  });
});
