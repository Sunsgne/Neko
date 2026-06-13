import { fileURLToPath } from "node:url";
import process from "node:process";

export function greet(name = "world"): string {
  return `Hello, ${name}! 🐱`;
}

function main(): void {
  console.log(greet());
}

const isMain = process.argv[1] === fileURLToPath(import.meta.url);
if (isMain) {
  main();
}
