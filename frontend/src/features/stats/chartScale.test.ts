import { describe, expect, it } from "vitest";

import { axisTicks, niceMax, percentOf } from "./chartScale";

describe("niceMax", () => {
  it.each([
    [0, 15],
    [14, 15],
    [31, 60],
    [60, 60],
    [100, 120],
    [1000, 1440],
    [1500, 1680],
  ])("%s -> %s", (input, expected) => {
    expect(niceMax(input)).toBe(expected);
  });
});

describe("axisTicks", () => {
  it("returns bottom, middle and top", () => {
    expect(axisTicks(120)).toEqual([0, 60, 120]);
  });
});

describe("percentOf", () => {
  it("scales and clamps", () => {
    expect(percentOf(30, 60)).toBe(50);
    expect(percentOf(90, 60)).toBe(100);
    expect(percentOf(-5, 60)).toBe(0);
    expect(percentOf(10, 0)).toBe(0);
  });
});
