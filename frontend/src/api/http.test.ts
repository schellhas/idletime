import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError, request } from "./http";

function mockFetch(response: Response | Error) {
  const fn = response instanceof Error
    ? vi.fn().mockRejectedValue(response)
    : vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fn);
  return fn;
}

afterEach(() => vi.unstubAllGlobals());

describe("request", () => {
  it("returns the parsed body on success", async () => {
    const fetchMock = mockFetch(Response.json([{ id: 1 }]));

    await expect(request("/categories")).resolves.toEqual([{ id: 1 }]);
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/categories", expect.anything());
  });

  it("maps the error envelope to an ApiError", async () => {
    mockFetch(
      Response.json(
        { error: { code: "duplicate_name", message: "already exists" } },
        { status: 409 },
      ),
    );

    await expect(request("/categories")).rejects.toMatchObject({
      status: 409,
      code: "duplicate_name",
      message: "already exists",
    });
  });

  it("falls back to a generic error for non-JSON failures", async () => {
    mockFetch(new Response("boom", { status: 502 }));

    await expect(request("/categories")).rejects.toMatchObject({ status: 502, code: "unknown_error" });
  });

  it("wraps network failures", async () => {
    mockFetch(new TypeError("failed to fetch"));

    const error = await request("/categories").catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ status: 0, code: "network_error" });
  });
});
