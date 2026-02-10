import { createChannel } from "./channel.js";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export default function register(api: any) {
  api.registerChannel({ plugin: createChannel(api) });
}
