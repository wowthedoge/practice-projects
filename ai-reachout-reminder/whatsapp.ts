import twilio from "twilio";

const accountSid = process.env.TWILIO_ACCOUNT_SID!;
const authToken = process.env.TWILIO_AUTH_TOKEN!;
const fromNumber = process.env.TWILIO_WHATSAPP_NUMBER!; // e.g. "whatsapp:+14155238886"

const client = twilio(accountSid, authToken);

export async function sendWhatsApp(to: string, body: string): Promise<string> {
  const message = await client.messages.create({
    from: fromNumber,
    to,
    body,
  });
  return message.sid;
}
