import { MongoClient, type Collection } from "mongodb";
import type { Connection } from "./types.js";

const MONGO_URL = process.env.MONGO_URL || "mongodb://localhost:27017";
const DB_NAME = "reachout";

let client: MongoClient;

export async function getCollection(): Promise<Collection<Connection>> {
  if (!client) {
    client = new MongoClient(MONGO_URL);
    await client.connect();
  }
  return client.db(DB_NAME).collection<Connection>("connections");
}

export async function loadConnections(): Promise<Connection[]> {
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

export async function closeDb(): Promise<void> {
  if (client) await client.close();
}
