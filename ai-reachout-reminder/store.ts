import { randomBytes } from "crypto";
import { MongoClient, type Collection } from "mongodb";
import type { Connection, Session } from "./types.js";

const MONGO_URL = process.env.MONGO_URL || "mongodb://localhost:27017";
const DB_NAME = "reachout";

let client: MongoClient;

async function getDb() {
  if (!client) {
    client = new MongoClient(MONGO_URL);
    await client.connect();
  }
  return client.db(DB_NAME);
}

export async function getCollection(): Promise<Collection<Connection>> {
  const db = await getDb();
  return db.collection<Connection>("connections");
}

// --- Connections ---

export async function loadConnections(owner: string): Promise<Connection[]> {
  const col = await getCollection();
  return col.find({ owner }).toArray();
}

export async function loadAllConnections(): Promise<Connection[]> {
  const col = await getCollection();
  return col.find().toArray();
}

export async function addConnection(
  connection: Omit<Connection, "_id">
): Promise<Connection> {
  const col = await getCollection();
  const result = await col.insertOne(connection as Connection);
  return { ...connection, _id: result.insertedId };
}

// --- Sessions ---

async function sessionsCol(): Promise<Collection<Session>> {
  const db = await getDb();
  const col = db.collection<Session>("sessions");
  await col.createIndex({ expiresAt: 1 }, { expireAfterSeconds: 0 });
  return col;
}

export async function createSession(phone: string): Promise<string> {
  const col = await sessionsCol();
  const token = randomBytes(32).toString("hex");
  const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000); // 7 days
  await col.insertOne({ token, phone, expiresAt });
  return token;
}

export async function getSession(token: string): Promise<Session | null> {
  const col = await sessionsCol();
  return col.findOne({ token, expiresAt: { $gt: new Date() } });
}

export async function deleteSession(token: string): Promise<void> {
  const col = await sessionsCol();
  await col.deleteOne({ token });
}

// --- Verification codes (in-memory, 5 min TTL) ---

const codes = new Map<string, { code: string; expires: number }>();

export function storeVerificationCode(phone: string, code: string): void {
  codes.set(phone, { code, expires: Date.now() + 5 * 60 * 1000 });
}

export function verifyCode(phone: string, code: string): boolean {
  const entry = codes.get(phone);
  if (!entry || entry.expires < Date.now() || entry.code !== code) return false;
  codes.delete(phone);
  return true;
}

export async function closeDb(): Promise<void> {
  if (client) await client.close();
}
