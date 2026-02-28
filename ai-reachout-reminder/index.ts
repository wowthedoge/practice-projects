import Anthropic from "@anthropic-ai/sdk";
import { loadConnections, closeDb } from "./store.js";
import { sendWhatsApp } from "./whatsapp.js";
import type { Connection, Reminder } from "./types.js";

// Find events that need a reachout today.
// Birthdays: message on the same day. Other events: message 3 days after.
function findReachouts(connections: Connection[]): Reminder[] {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  const reminders: Reminder[] = [];

  for (const connection of connections) {
    for (const event of connection.events) {
      const eventDate = new Date(event.date);
      const eventDay = new Date(
        eventDate.getFullYear(),
        eventDate.getMonth(),
        eventDate.getDate()
      );
      const daysSince = Math.floor(
        (today.getTime() - eventDay.getTime()) / (1000 * 60 * 60 * 24)
      );

      const isBirthday = event.name.toLowerCase() === "birthday";
      const shouldReach = isBirthday ? daysSince === 0 : daysSince === 3;

      if (shouldReach) {
        reminders.push({ connection, event, daysSince });
      }
    }
  }

  return reminders;
}

// Generate a personalized message using Claude
async function craftMessage(
  client: Anthropic,
  reminder: Reminder
): Promise<string> {
  const { connection, event } = reminder;

  const prompt = `Craft a short, warm, personalized text message to a connection.

Connection: ${connection.name}
Context about them: ${connection.summary}
Event: ${event.name} on ${new Date(event.date).toLocaleDateString("en-US", { weekday: "long", month: "long", day: "numeric" })}${reminder.daysSince > 0 ? ` (${reminder.daysSince} days ago)` : " (today)"}
${event.summary ? `Event details: ${event.summary}` : ""}

Write a brief, natural text message (2-3 sentences max). ${reminder.daysSince === 0 ? "Wish them well on their special day." : "Follow up after the event — ask how it went, reference something specific."} The tone should be casual and genuine, not salesy. Do not include any subject line or greeting like "Hi [name]" — just the message body.`;

  const response = await client.messages.create({
    model: "claude-haiku-4-5",
    max_tokens: 256,
    messages: [{ role: "user", content: prompt }],
  });

  return response.content[0].type === "text" ? response.content[0].text : "";
}

// Main
async function main() {
  const connections = await loadConnections();
  const reminders = findReachouts(connections);

  if (reminders.length === 0) {
    console.log("No reachouts needed today.");
    return;
  }

  console.log(`\n📋 ${reminders.length} reachout(s) for today\n`);
  console.log("─".repeat(60));

  const client = new Anthropic();

  for (const reminder of reminders) {
    const { connection, event, daysSince } = reminder;
    const dateStr = new Date(event.date).toLocaleDateString("en-US", {
      weekday: "long",
      month: "long",
      day: "numeric",
    });

    console.log(`\n👤 ${connection.name}  📱 ${connection.phone}`);
    console.log(
      `📅 ${event.name} — ${dateStr} (${daysSince === 0 ? "today" : `${daysSince} days ago`})`
    );

    const message = await craftMessage(client, reminder);
    console.log(`\n💬 Suggested message:\n   "${message}"`);

    if (connection.phone) {
      const to = `whatsapp:${connection.phone.startsWith("+") ? "" : "+"}${connection.phone}`;
      try {
        const sid = await sendWhatsApp(to, message);
        console.log(`✅ Sent via WhatsApp (${sid})`);
      } catch (err) {
        console.error(`❌ WhatsApp send failed:`, err);
      }
    } else {
      console.log("⚠️  No phone number — skipping WhatsApp");
    }

    console.log("\n" + "─".repeat(60));
  }
}

main()
  .catch(console.error)
  .finally(() => closeDb());
