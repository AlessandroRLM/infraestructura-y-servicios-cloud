import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readPreferredArea, writePreferredArea } from "../areaPreference";

// happy-dom >= 17 requires --localstorage-file for a persistent store.
// For unit tests we use an in-memory stub so the test file needs no external
// file path and remains fully self-contained.
function makeLocalStorageStub(): Storage {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
    key: (index: number) => Object.keys(store)[index] ?? null,
    get length() {
      return Object.keys(store).length;
    },
  };
}

describe("readPreferredArea", () => {
  let storageStub: Storage;

  beforeEach(() => {
    storageStub = makeLocalStorageStub();
    vi.stubGlobal("localStorage", storageStub);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns null when the key is absent", () => {
    expect(readPreferredArea()).toBeNull();
  });

  it("returns 'admin' when the stored value is 'admin'", () => {
    storageStub.setItem("iyc.preferredArea", "admin");
    expect(readPreferredArea()).toBe("admin");
  });

  it("returns 'participant' when the stored value is 'participant'", () => {
    storageStub.setItem("iyc.preferredArea", "participant");
    expect(readPreferredArea()).toBe("participant");
  });

  it("returns null when the stored value is an unknown string", () => {
    storageStub.setItem("iyc.preferredArea", "superadmin");
    expect(readPreferredArea()).toBeNull();
  });

  it("returns null and does not throw when localStorage.getItem throws", () => {
    vi.stubGlobal("localStorage", {
      ...storageStub,
      getItem: () => {
        throw new Error("storage unavailable");
      },
    });
    expect(() => readPreferredArea()).not.toThrow();
    expect(readPreferredArea()).toBeNull();
  });
});

describe("writePreferredArea", () => {
  let storageStub: Storage;

  beforeEach(() => {
    storageStub = makeLocalStorageStub();
    vi.stubGlobal("localStorage", storageStub);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("writes 'admin' to localStorage", () => {
    writePreferredArea("admin");
    expect(storageStub.getItem("iyc.preferredArea")).toBe("admin");
  });

  it("writes 'participant' to localStorage", () => {
    writePreferredArea("participant");
    expect(storageStub.getItem("iyc.preferredArea")).toBe("participant");
  });

  it("does not throw when localStorage.setItem throws", () => {
    vi.stubGlobal("localStorage", {
      ...storageStub,
      setItem: () => {
        throw new Error("quota exceeded");
      },
    });
    expect(() => writePreferredArea("admin")).not.toThrow();
  });
});
