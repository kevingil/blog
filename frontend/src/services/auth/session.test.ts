import { afterAll, beforeAll, beforeEach, describe, expect, test } from "bun:test";
import { getDefaultStore } from "jotai";
import type { User } from "../types";

type SessionModule = typeof import("./session");

class MemoryStorage {
  private values = new Map<string, string>();
  getItem(key: string) {
    return this.values.get(key) ?? null;
  }
  setItem(key: string, value: string) {
    this.values.set(key, value);
  }
  removeItem(key: string) {
    this.values.delete(key);
  }
  clear() {
    this.values.clear();
  }
}

function token(claims: Record<string, unknown>): string {
  const encode = (value: unknown) =>
    btoa(JSON.stringify(value)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  return `${encode({ alg: "HS256", typ: "JWT" })}.${encode(claims)}.signature`;
}

const user: User = { id: 1, name: "Ada", email: "ada@example.test", role: "admin" };
const storage = new MemoryStorage();
const previousWindow = globalThis.window;
const previousStorage = globalThis.localStorage;
let session: SessionModule;

beforeAll(async () => {
  globalThis.window = globalThis as Window & typeof globalThis;
  globalThis.localStorage = storage as unknown as Storage;
  session = await import("./session");
});

afterAll(() => {
  globalThis.window = previousWindow;
  globalThis.localStorage = previousStorage;
});

beforeEach(() => {
  session.endSession();
  storage.clear();
});

describe("sessionFromToken", () => {
  test("reads issue and expiry times from the token claims", () => {
    const parsed = session.sessionFromToken(token({ iat: 1_000, exp: 2_000 }), user);
    expect(parsed).toMatchObject({ issuedAt: 1_000_000, expiresAt: 2_000_000, user });
  });

  test("rejects tokens without a usable expiry", () => {
    expect(session.sessionFromToken(token({ iat: 1_000 }), user)).toBeNull();
    expect(session.sessionFromToken("not-a-jwt", user)).toBeNull();
    expect(session.sessionFromToken("a.%%%.c", user)).toBeNull();
  });
});

describe("session lifetime", () => {
  test("a saved session persists until it expires, then reading it signs out", () => {
    const now = Date.now();
    const saved = session.saveSession(token({ iat: now / 1000, exp: now / 1000 + 60 }), user);

    expect(session.currentSession(now)).toEqual(saved);
    expect(storage.getItem("user")).toBe(JSON.stringify(user));

    expect(session.currentSession(now + 61_000)).toBeNull();
    expect(getDefaultStore().get(session.sessionAtom)).toBeNull();
    expect(storage.getItem("token")).toBeNull();
  });

  test("adopts sign-in and sign-out from another tab and ignores expired tokens", () => {
    const now = Math.floor(Date.now() / 1000);
    const fresh = token({ iat: now, exp: now + 3_600 });
    storage.setItem("token", fresh);
    storage.setItem("user", JSON.stringify(user));
    session.syncSessionFromStorage();
    expect(session.currentSession()?.token).toBe(fresh);

    storage.setItem("token", token({ iat: now - 7_200, exp: now - 3_600 }));
    session.syncSessionFromStorage();
    expect(session.currentSession()).toBeNull();

    storage.setItem("token", fresh);
    session.syncSessionFromStorage();
    storage.clear();
    session.syncSessionFromStorage();
    expect(session.currentSession()).toBeNull();
  });
});
