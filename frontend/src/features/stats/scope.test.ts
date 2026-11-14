import { describe, expect, it } from "vitest";

import { activityScopeKey, categoryScopeKey, parseScopeKey } from "./scope";

describe("scope keys", () => {
  it("round-trips categories and activities", () => {
    expect(parseScopeKey(categoryScopeKey(3))).toEqual({ categoryId: 3 });
    expect(parseScopeKey(activityScopeKey(7))).toEqual({ activityId: 7 });
  });

  it("treats an empty or malformed key as no scope", () => {
    expect(parseScopeKey("")).toEqual({});
    expect(parseScopeKey("c:abc")).toEqual({});
    expect(parseScopeKey("x:1")).toEqual({});
  });
});
