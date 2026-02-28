import type { ObjectId } from "mongodb";

export interface ConnectionEvent {
  date: string;
  name: string;
  summary: string;
}

export interface Connection {
  _id?: ObjectId;
  name: string;
  phone: string | null;
  summary: string;
  events: ConnectionEvent[];
}

export interface Reminder {
  connection: Connection;
  event: ConnectionEvent;
  daysSince: number;
}
