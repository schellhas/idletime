import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError, http, onUnauthenticated } from "./http";

function mockFetch(response: Response | Error) {
  const fn = response instanceof Error
    ? vi.fn().mockRejectedValue(response)
    : vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fn);
  return fn;
}

afterEach(() => vi.unstubAllGlobals());

describe("http", () => {
  it("returns the parsed body on success", async () => {
    const fetchMock = mockFetch(Response.json([{ id: 1 }]));

    await expect(http.get("/categories")).resolves.toEqual([{ id: 1 }]);
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/categories", expect.objectContaining({ method: "GET" }));
  });

  it("sends JSON bodies with a content type", async () => {
    const fetchMock = mockFetch(new Response(null, { status: 202 }));

    await http.post("/auth/register", { username: "alice" });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/auth/register",
      expect.objectContaining({
        method: "POST",
        body: '{"username":"alice"}',
        headers: { "Content-Type": "application/json" },
      }),
    );
  });

  it("accepts empty responses", async () => {
    mockFetch(new Response(null, { status: 204 }));

    await expect(http.post("/auth/logout")).resolves.toBeUndefined();
  });

  it("maps the error envelope to an ApiError", async () => {
    mockFetch(
      Response.json(
        { error: { code: "duplicate_name", message: "already exists" } },
        { status: 409 },
      ),
    );

    await expect(http.get("/categories")).rejects.toMatchObject({
      status: 409,
      code: "duplicate_name",
      message: "already exists",
    });
  });

  it("falls back to a generic error for non-JSON failures", async () => {
    mockFetch(new Response("boom", { status: 502 }));

    await expect(http.get("/categories")).rejects.toMatchObject({ status: 502, code: "unknown_error" });
  });

  it("wraps network failures", async () => {
    mockFetch(new TypeError("failed to fetch"));

    const error = await http.get("/categories").catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ status: 0, code: "network_error" });
  });
});

describe("session expiry signal", () => {
  it("fires on a 401 from a normal request", async () => {
    mockFetch(Response.json({ error: { code: "unauthenticated" } }, { status: 401 }));
    const listener = vi.fn();
    const unsubscribe = onUnauthenticated(listener);

    await http.get("/me").catch(() => undefined);

    expect(listener).toHaveBeenCalledOnce();
    unsubscribe();
  });

  it("does not fire for failed logins", async () => {
    mockFetch(Response.json({ error: { code: "invalid_credentials" } }, { status: 401 }));
    const listener = vi.fn();
    const unsubscribe = onUnauthenticated(listener);

    await http.post("/auth/login", {}).catch(() => undefined);

    expect(listener).not.toHaveBeenCalled();
    unsubscribe();
  });
});
