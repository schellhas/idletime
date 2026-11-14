import { afterEach, describe, expect, it, vi } from "vitest";

import { getRecommendation, recommendationQuery } from "./progress";

afterEach(() => vi.unstubAllGlobals());

describe("recommendationQuery", () => {
  it("is empty without options", () => {
    expect(recommendationQuery({})).toBe("");
  });

  it("encodes all options", () => {
    expect(recommendationQuery({ availableMinutes: 30, categoryId: 2, skip: [4, 5] })).toBe(
      "?availableMinutes=30&categoryId=2&skip=4%2C5",
    );
  });

  it("treats zero minutes and an empty skip list as unset", () => {
    expect(recommendationQuery({ availableMinutes: 0, skip: [] })).toBe("");
  });
});

describe("getRecommendation", () => {
  it("returns null when nothing fits", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(Response.json({ error: { code: "no_recommendation", message: "x" } }, { status: 404 })),
    );

    await expect(getRecommendation({})).resolves.toBeNull();
  });

  it("rethrows other errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(Response.json({ error: { code: "category_not_found", message: "x" } }, { status: 404 })),
    );

    await expect(getRecommendation({ categoryId: 9 })).rejects.toMatchObject({ code: "category_not_found" });
  });
});
