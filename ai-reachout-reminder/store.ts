import { readFileSync, writeFileSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { randomUUID } from "crypto";
import type { Connection } from "./types.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const CONNECTIONS_PATH = resolve(__dirname, "connections.json");

export function loadConnections(): Connection[] {
  return JSON.parse(readFileSync(CONNECTIONS_PATH, "utf-8"));
}

export function saveConnections(connections: Connection[]): void {
  writeFileSync(CONNECTIONS_PATH, JSON.stringify(connections, null, 2) + "\n");
}

export function addConnection(
  connection: Omit<Connection, "id">
): Connection {
  const connections = loadConnections();
  const newConnection: Connection = {
    id: randomUUID(),
    ...connection,
  };
  connections.push(newConnection);
  saveConnections(connections);
  return newConnection;
}
