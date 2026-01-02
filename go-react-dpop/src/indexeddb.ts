const DB_NAME = "dpop-auth";
const STORE_NAME = "dpop-keys";
const KEY_ID = "dpop-keypair"

export function saveKeyPair(keyPair: CryptoKeyPair): Promise<void> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open("dpop-auth");
    request.onupgradeneeded = () =>
      request.result.createObjectStore(STORE_NAME);
    request.onerror = () => reject(request.error);
    request.onsuccess = () => {
      const tx = request.result.transaction(STORE_NAME, "readwrite");
      tx.objectStore(STORE_NAME).put(keyPair, KEY_ID);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    };
  });
}
