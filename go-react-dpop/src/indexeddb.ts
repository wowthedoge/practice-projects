import { openDB, type IDBPDatabase } from "idb";

const DB_NAME = "dpop-auth";
const STORE_NAME = "dpop-keys";
const KEY_ID = "dpop-keypair";

function getDB(): Promise<IDBPDatabase> {
  return openDB(DB_NAME, 1, {
    upgrade(db) {
      db.createObjectStore(STORE_NAME);
    },
  });
}

export async function saveKeyPair(keyPair: CryptoKeyPair): Promise<void> {
  const db = await getDB();
  await db.put(STORE_NAME, keyPair, KEY_ID);
}

export async function getKeyPair(): Promise<CryptoKeyPair | undefined> {
  const db = await getDB();
  const keyPair = await db.get(STORE_NAME, KEY_ID);
  if (!keyPair) return undefined;
  return keyPair;
}
