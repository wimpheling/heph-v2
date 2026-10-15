import { useEffect, useState } from "react";
import { AppFrame, Button, Heading, Panel, Stack, Status, Text } from "@heph/design-system";
import { getSystemInfo } from "./app/rpc/system";

type Connection =
  | { state: "connecting" }
  | { state: "connected"; name: string; version: string }
  | { state: "unavailable" };

export function App() {
  const [attempt, setAttempt] = useState(0);
  const [connection, setConnection] = useState<Connection>({ state: "connecting" });

  useEffect(() => {
    const controller = new AbortController();
    setConnection({ state: "connecting" });
    getSystemInfo(controller.signal).then(
      (info) => {
        if (!controller.signal.aborted)
          setConnection({ state: "connected", name: info.name, version: info.version });
      },
      () => {
        if (!controller.signal.aborted) setConnection({ state: "unavailable" });
      },
    );
    return () => controller.abort();
  }, [attempt]);

  return (
    <AppFrame>
      <Panel>
        <Stack>
          <Text>heph-v2 / bootstrap</Text>
          <Heading>A home for code and the agents that build it.</Heading>
          <Text>The foundation is ready. Repositories and agent execution come next.</Text>
          <Status
            tone={
              connection.state === "connected"
                ? "success"
                : connection.state === "unavailable"
                  ? "error"
                  : "neutral"
            }
          >
            {connection.state === "connected"
              ? `Connected to ${connection.name} · v${connection.version}`
              : connection.state === "unavailable"
                ? "Server unavailable. Start the Rust server and retry."
                : "Connecting to the Rust server…"}
          </Status>
          <Button
            label="Check connection"
            variant="secondary"
            isDisabled={connection.state === "connecting"}
            onClick={() => setAttempt((value) => value + 1)}
          />
        </Stack>
      </Panel>
    </AppFrame>
  );
}
