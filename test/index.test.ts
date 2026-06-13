import assert from "node:assert/strict";
import { test } from "node:test";
import { greet } from "../src/index.ts";

test("greet uses default name", () => {
  assert.equal(greet(), "Hello, world! 🐱");
});

test("greet uses provided name", () => {
  assert.equal(greet("Neko"), "Hello, Neko! 🐱");
});
